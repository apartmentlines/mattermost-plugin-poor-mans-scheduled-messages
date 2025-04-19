package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockCommand struct {
	UserDeleteMessageFunc  func(userID, msgID string) (*types.ScheduledMessage, error)
	BuildEphemeralListFunc func(args *model.CommandArgs) *model.CommandResponse
}

func (m *mockCommand) Register() error { panic("not implemented") } // Not needed by api.go
func (m *mockCommand) Execute(*model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	panic("not implemented") // Not needed by api.go
}
func (m *mockCommand) UserDeleteMessage(userID, msgID string) (*types.ScheduledMessage, error) {
	if m.UserDeleteMessageFunc != nil {
		return m.UserDeleteMessageFunc(userID, msgID)
	}
	panic("UserDeleteMessageFunc not set")
}
func (m *mockCommand) BuildEphemeralList(args *model.CommandArgs) *model.CommandResponse {
	if m.BuildEphemeralListFunc != nil {
		return m.BuildEphemeralListFunc(args)
	}
	panic("BuildEphemeralListFunc not set")
}

func setupPluginForAPI(t *testing.T, ctrl *gomock.Controller) (*Plugin, *mock.MockPostService, *mock.MockChannelService, *mockCommand) {
	t.Helper()
	postMock := mock.NewMockPostService(ctrl)
	channelMock := mock.NewMockChannelService(ctrl)
	cmdMock := &mockCommand{}

	p := &Plugin{
		logger:  &testutil.FakeLogger{},
		poster:  postMock,
		Channel: channelMock,
		Command: cmdMock,
	}
	return p, postMock, channelMock, cmdMock
}

func createDeleteRequest(t *testing.T, userID, postID, channelID, action, msgID string) *http.Request {
	t.Helper()
	reqBody := model.PostActionIntegrationRequest{
		PostId:    postID,
		ChannelId: channelID,
		Context: map[string]any{
			"action": action,
			"id":     msgID,
		},
	}
	b, err := json.Marshal(reqBody)
	require.NoError(t, err)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/delete", bytes.NewReader(b))
	if userID != "" {
		r.Header.Set(constants.HTTPHeaderMattermostUserID, userID)
	}
	return r
}

func TestMattermostAuthorizationRequired_Unauthorized(t *testing.T) {
	p := &Plugin{logger: &testutil.FakeLogger{}}
	handlerCalled := false
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil) // No MM User ID header
	rr := httptest.NewRecorder()

	p.MattermostAuthorizationRequired(dummyHandler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Equal(t, "Not authorized\n", rr.Body.String())
	assert.False(t, handlerCalled, "Wrapped handler should not have been called")
}

func TestMattermostAuthorizationRequired_Authorized(t *testing.T) {
	p := &Plugin{logger: &testutil.FakeLogger{}}
	handlerCalled := false
	dummyHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(constants.HTTPHeaderMattermostUserID, "test-user-id")
	rr := httptest.NewRecorder()

	p.MattermostAuthorizationRequired(dummyHandler).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, handlerCalled, "Wrapped handler should have been called")
}

func TestParseDeleteRequest_MalformedJSON(t *testing.T) {
	p := &Plugin{logger: &testutil.FakeLogger{}}
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("{invalid json"))
	_, _, err := parseDeleteRequest(p, r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid request body")
}

func TestParseDeleteRequest_MissingContextFields(t *testing.T) {
	tests := []struct {
		name    string
		context map[string]any
	}{
		{"Missing action", map[string]any{"id": "msg123"}},
		{"Missing id", map[string]any{"action": "delete"}},
		{"Nil context", nil},
		{"Empty context", map[string]any{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &Plugin{logger: &testutil.FakeLogger{}}
			reqBody := model.PostActionIntegrationRequest{Context: tc.context}
			b, _ := json.Marshal(reqBody)
			r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
			_, _, err := parseDeleteRequest(p, r)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid delete request context")
		})
	}
}

func TestParseDeleteRequest_WrongAction(t *testing.T) {
	p := &Plugin{logger: &testutil.FakeLogger{}}
	reqBody := model.PostActionIntegrationRequest{Context: map[string]any{"action": "not-delete", "id": "msg123"}}
	b, _ := json.Marshal(reqBody)
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	_, _, err := parseDeleteRequest(p, r)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid delete request context")
}

func TestParseDeleteRequest_Valid(t *testing.T) {
	p := &Plugin{logger: &testutil.FakeLogger{}}
	reqBody := model.PostActionIntegrationRequest{
		PostId:    "post1",
		ChannelId: "chan1",
		Context:   map[string]any{"action": "delete", "id": "msg123"},
	}
	b, _ := json.Marshal(reqBody)
	r := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req, msgID, err := parseDeleteRequest(p, r)
	require.NoError(t, err)
	assert.Equal(t, "post1", req.PostId)
	assert.Equal(t, "chan1", req.ChannelId)
	assert.Equal(t, "msg123", msgID)
}

func TestServeHTTP_Delete_AuthFail(t *testing.T) { // TC-3.1
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p, _, _, _ := setupPluginForAPI(t, ctrl)

	req := createDeleteRequest(t, "", "post1", "chan1", "delete", "msg1")
	rr := httptest.NewRecorder()

	p.ServeHTTP(nil, rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.Contains(t, rr.Body.String(), "Not authorized")
}

func TestServeHTTP_Delete_BadRequestBody(t *testing.T) { // TC-3.2
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p, _, _, _ := setupPluginForAPI(t, ctrl)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/delete", strings.NewReader("{bad json"))
	req.Header.Set(constants.HTTPHeaderMattermostUserID, "u1")
	rr := httptest.NewRecorder()

	p.ServeHTTP(nil, rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid request body")
}

func TestServeHTTP_Delete_InvalidContext(t *testing.T) { // TC-3.3
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p, _, _, _ := setupPluginForAPI(t, ctrl)

	req := createDeleteRequest(t, "u1", "post1", "chan1", "wrong-action", "msg1")
	rr := httptest.NewRecorder()

	p.ServeHTTP(nil, rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "invalid delete request context")
}

func TestServeHTTP_Delete_CommandLayerFailure(t *testing.T) { // TC-3.4
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p, postMock, _, cmdMock := setupPluginForAPI(t, ctrl)

	cmdMock.UserDeleteMessageFunc = func(userID, msgID string) (*types.ScheduledMessage, error) {
		assert.Equal(t, "u1", userID)
		assert.Equal(t, "msg999", msgID)
		return nil, errors.New("command layer boom")
	}
	// Ensure poster methods are NOT called
	postMock.EXPECT().UpdateEphemeralPost(gomock.Any(), gomock.Any()).Times(0)
	postMock.EXPECT().SendEphemeralPost(gomock.Any(), gomock.Any()).Times(0)

	req := createDeleteRequest(t, "u1", "ephemeral123", "chanABC", "delete", "msg999")
	rr := httptest.NewRecorder()

	p.ServeHTTP(nil, rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.Contains(t, rr.Body.String(), "Failed to delete message: command layer boom")
}

func TestServeHTTP_Delete_HappyPath_NormalTimezone(t *testing.T) { // TC-3.5
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p, postMock, channelMock, cmdMock := setupPluginForAPI(t, ctrl)

	userID := "u1"
	msgID := "msg999"
	postID := "ephemeral123"
	channelID := "chanABC"           // Request channel
	deletedMsgChannelID := "chanDEF" // Channel where message was scheduled
	expectedTime := time.Date(2025, 1, 2, 15, 4, 0, 0, time.UTC)
	expectedTimezone := "UTC"
	expectedChannelLink := "~town-square"
	expectedAttachments := []any{"att1"}

	// Command Mock Setup
	cmdMock.UserDeleteMessageFunc = func(u, id string) (*types.ScheduledMessage, error) {
		assert.Equal(t, userID, u)
		assert.Equal(t, msgID, id)
		return &types.ScheduledMessage{
			ID:        msgID,
			UserID:    userID,
			ChannelID: deletedMsgChannelID,
			PostAt:    expectedTime,
			Timezone:  expectedTimezone,
		}, nil
	}
	cmdMock.BuildEphemeralListFunc = func(args *model.CommandArgs) *model.CommandResponse {
		assert.Equal(t, userID, args.UserId)
		return &model.CommandResponse{Props: map[string]any{"attachments": expectedAttachments}}
	}

	// Channel Mock Setup
	channelMock.EXPECT().GetInfoOrUnknown(deletedMsgChannelID).Return(&ports.ChannelInfo{ChannelID: deletedMsgChannelID})
	channelMock.EXPECT().MakeChannelLink(gomock.Any()).DoAndReturn(func(info *ports.ChannelInfo) string {
		assert.Equal(t, deletedMsgChannelID, info.ChannelID)
		return expectedChannelLink
	})

	// Post Mock Setup
	postMock.EXPECT().UpdateEphemeralPost(userID, gomock.Any()).Do(func(_ string, post *model.Post) {
		assert.Equal(t, postID, post.Id)
		assert.Equal(t, userID, post.UserId)
		assert.Equal(t, channelID, post.ChannelId)
		assert.Equal(t, expectedAttachments, post.Props["attachments"])
	})
	postMock.EXPECT().SendEphemeralPost(userID, gomock.Any()).Do(func(_ string, post *model.Post) {
		assert.Equal(t, userID, post.UserId)
		assert.Equal(t, channelID, post.ChannelId)
		expectedTimeStr := expectedTime.Format(constants.TimeLayout)
		expectedMsg := fmt.Sprintf("%s Message scheduled for **%s** %s has been deleted.", constants.EmojiSuccess, expectedTimeStr, expectedChannelLink)
		assert.Equal(t, expectedMsg, post.Message)
	})

	// Execute
	req := createDeleteRequest(t, userID, postID, channelID, "delete", msgID)
	rr := httptest.NewRecorder()
	p.ServeHTTP(nil, rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "", rr.Body.String()) // No body on success
}

func TestServeHTTP_Delete_HappyPath_InvalidTimezoneFallback(t *testing.T) { // TC-3.6
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p, postMock, channelMock, cmdMock := setupPluginForAPI(t, ctrl)

	userID := "u1"
	msgID := "msg999"
	postID := "ephemeral123"
	channelID := "chanABC"
	deletedMsgChannelID := "chanDEF"
	expectedTime := time.Date(2025, 1, 2, 15, 4, 0, 0, time.UTC) // Still UTC
	invalidTimezone := "Not/A/Zone"
	expectedChannelLink := "~town-square"

	// Command Mock Setup (only difference is the timezone)
	cmdMock.UserDeleteMessageFunc = func(u, id string) (*types.ScheduledMessage, error) {
		return &types.ScheduledMessage{
			ID:        msgID,
			UserID:    userID,
			ChannelID: deletedMsgChannelID,
			PostAt:    expectedTime,
			Timezone:  invalidTimezone,
		}, nil
	}
	cmdMock.BuildEphemeralListFunc = func(args *model.CommandArgs) *model.CommandResponse {
		return &model.CommandResponse{Props: map[string]any{"attachments": []any{"att1"}}}
	}

	// Channel Mock Setup
	channelMock.EXPECT().GetInfoOrUnknown(deletedMsgChannelID).Return(&ports.ChannelInfo{ChannelID: deletedMsgChannelID})
	channelMock.EXPECT().MakeChannelLink(gomock.Any()).Return(expectedChannelLink)

	// Post Mock Setup
	postMock.EXPECT().UpdateEphemeralPost(userID, gomock.Any()) // Check details implicitly via TC-3.5
	postMock.EXPECT().SendEphemeralPost(userID, gomock.Any()).Do(func(_ string, post *model.Post) {
		// Expect UTC time format because the timezone was invalid
		expectedTimeStr := expectedTime.Format(constants.TimeLayout) // Format in UTC
		expectedMsg := fmt.Sprintf("%s Message scheduled for **%s** %s has been deleted.", constants.EmojiSuccess, expectedTimeStr, expectedChannelLink)
		assert.Equal(t, expectedMsg, post.Message)
	})

	// Execute
	req := createDeleteRequest(t, userID, postID, channelID, "delete", msgID)
	rr := httptest.NewRecorder()
	p.ServeHTTP(nil, rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestUpdateEphemeralPostWithList(t *testing.T) { // TC-4.1
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p, postMock, _, _ := setupPluginForAPI(t, ctrl)

	userID := "user123"
	postID := "post456"
	channelID := "chan789"
	attachments := []any{"attachment_data"}
	updatedList := &model.CommandResponse{
		Props: map[string]any{"attachments": attachments},
	}

	postMock.EXPECT().UpdateEphemeralPost(userID, gomock.Any()).Do(func(_ string, post *model.Post) {
		assert.Equal(t, postID, post.Id)
		assert.Equal(t, userID, post.UserId)
		assert.Equal(t, channelID, post.ChannelId)
		assert.Equal(t, attachments, post.Props["attachments"])
	})

	p.updateEphemeralPostWithList(userID, postID, channelID, updatedList)
}

func TestSendDeletionConfirmation_NormalTimezone(t *testing.T) { // TC-5.1
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p, postMock, channelMock, _ := setupPluginForAPI(t, ctrl)

	userID := "user1"
	channelID := "chan1"                            // Channel where confirmation is sent
	deletedMsgChannelID := "chan2"                  // Channel where message was scheduled
	loc, _ := time.LoadLocation("America/New_York") // EST/EDT
	postAt := time.Date(2024, 7, 4, 10, 30, 0, 0, loc)
	deletedMsg := &types.ScheduledMessage{
		ID:             "msg1",
		UserID:         userID,
		ChannelID:      deletedMsgChannelID,
		PostAt:         postAt,
		Timezone:       "America/New_York",
		MessageContent: "test message",
	}
	expectedChannelLink := "in channel ~some-channel"

	channelMock.EXPECT().GetInfoOrUnknown(deletedMsgChannelID).Return(&ports.ChannelInfo{ChannelID: deletedMsgChannelID})
	channelMock.EXPECT().MakeChannelLink(gomock.Any()).Return(expectedChannelLink)

	postMock.EXPECT().SendEphemeralPost(userID, gomock.Any()).Do(func(_ string, post *model.Post) {
		assert.Equal(t, userID, post.UserId)
		assert.Equal(t, channelID, post.ChannelId)
		expectedTimeStr := postAt.Format(constants.TimeLayout)
		expectedMsg := fmt.Sprintf("%s Message scheduled for **%s** %s has been deleted.", constants.EmojiSuccess, expectedTimeStr, expectedChannelLink)
		assert.Equal(t, expectedMsg, post.Message)
	})

	p.sendDeletionConfirmation(userID, channelID, deletedMsg)
}

func TestSendDeletionConfirmation_InvalidTimezoneFallback(t *testing.T) { // TC-5.2
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	p, postMock, channelMock, _ := setupPluginForAPI(t, ctrl)

	userID := "user1"
	channelID := "chan1"
	deletedMsgChannelID := "chan2"
	postAt := time.Date(2024, 7, 4, 10, 30, 0, 0, time.UTC) // Use UTC time
	deletedMsg := &types.ScheduledMessage{
		ID:             "msg1",
		UserID:         userID,
		ChannelID:      deletedMsgChannelID,
		PostAt:         postAt,
		Timezone:       "Invalid/Timezone", // Bad timezone
		MessageContent: "test message",
	}
	expectedChannelLink := "in channel ~some-channel"

	channelMock.EXPECT().GetInfoOrUnknown(deletedMsgChannelID).Return(&ports.ChannelInfo{ChannelID: deletedMsgChannelID})
	channelMock.EXPECT().MakeChannelLink(gomock.Any()).Return(expectedChannelLink)

	postMock.EXPECT().SendEphemeralPost(userID, gomock.Any()).Do(func(_ string, post *model.Post) {
		assert.Equal(t, userID, post.UserId)
		assert.Equal(t, channelID, post.ChannelId)
		expectedTimeStr := postAt.Format(constants.TimeLayout) // Format in UTC
		expectedMsg := fmt.Sprintf("%s Message scheduled for **%s** %s has been deleted.", constants.EmojiSuccess, expectedTimeStr, expectedChannelLink)
		assert.Equal(t, expectedMsg, post.Message)
	})

	p.sendDeletionConfirmation(userID, channelID, deletedMsg)
}
