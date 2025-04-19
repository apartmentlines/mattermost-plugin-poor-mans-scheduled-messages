package command

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/formatter"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestMessage(id string, userID string, channelID string, content string, timezone string, postAt time.Time) *types.ScheduledMessage {
	return &types.ScheduledMessage{
		ID:             id,
		UserID:         userID,
		ChannelID:      channelID,
		MessageContent: content,
		Timezone:       timezone,
		PostAt:         postAt,
	}
}

func TestNewListService_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)
	logger := testutil.FakeLogger{}

	service := NewListService(logger, mockStore, mockChannel)

	require.NotNil(t, service)
	assert.Equal(t, logger, service.logger)
	assert.Equal(t, mockStore, service.store)
	assert.Equal(t, mockChannel, service.channel)
}

func TestBuild_LoadMessagesError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)
	logger := testutil.FakeLogger{}
	service := NewListService(logger, mockStore, mockChannel)
	userID := "user1"
	expectedErr := errors.New("store error")

	mockStore.EXPECT().ListUserMessageIDs(userID).Return(nil, expectedErr)

	response := service.Build(userID)

	expectedResponse := errorResponse(fmt.Sprintf("%s Error retrieving message list: %v", constants.EmojiError, expectedErr))
	assert.Equal(t, expectedResponse, response)
}

func TestBuild_NoMessages(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)
	logger := testutil.FakeLogger{}
	service := NewListService(logger, mockStore, mockChannel)
	userID := "user1"

	mockStore.EXPECT().ListUserMessageIDs(userID).Return([]string{}, nil)

	response := service.Build(userID)

	expectedResponse := emptyResponse()
	assert.Equal(t, expectedResponse, response)
}

func TestBuild_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)
	logger := testutil.FakeLogger{}
	service := NewListService(logger, mockStore, mockChannel)

	userID := "user1"
	now := time.Now()
	msg1 := createTestMessage("id1", userID, "ch1", "msg content 1", "UTC", now.Add(1*time.Hour))
	msg2 := createTestMessage("id2", userID, "ch2", "msg content 2", "UTC", now.Add(2*time.Hour))
	info1 := &ports.ChannelInfo{ChannelID: "ch1", ChannelType: model.ChannelTypeOpen, ChannelLink: "~town-square", TeamName: "team1"}
	info2 := &ports.ChannelInfo{ChannelID: "ch2", ChannelType: model.ChannelTypePrivate, ChannelLink: "~private-channel", TeamName: "team1"}

	mockStore.EXPECT().ListUserMessageIDs(userID).Return([]string{"id1", "id2"}, nil)
	mockStore.EXPECT().GetScheduledMessage("id1").Return(msg1, nil)
	mockStore.EXPECT().GetScheduledMessage("id2").Return(msg2, nil)
	mockChannel.EXPECT().GetInfoOrUnknown("ch1").Return(info1)
	mockChannel.EXPECT().GetInfoOrUnknown("ch2").Return(info2)
	mockChannel.EXPECT().MakeChannelLink(info1).Return("in channel: ~town-square")
	mockChannel.EXPECT().MakeChannelLink(info2).Return("in channel: ~private-channel")

	response := service.Build(userID)

	require.NotNil(t, response)
	assert.Equal(t, model.CommandResponseTypeEphemeral, response.ResponseType)
	assert.Equal(t, constants.ListHeader, response.Text)
	require.NotNil(t, response.Props)
	attachments, ok := response.Props["attachments"].([]*model.SlackAttachment)
	require.True(t, ok)
	require.Len(t, attachments, 2)

	loc, _ := time.LoadLocation("UTC")
	expectedHeader1 := formatter.FormatListAttachmentHeader(msg1.PostAt.In(loc), "in channel: ~town-square", msg1.MessageContent)
	expectedHeader2 := formatter.FormatListAttachmentHeader(msg2.PostAt.In(loc), "in channel: ~private-channel", msg2.MessageContent)

	assert.Equal(t, expectedHeader1, attachments[0].Text)
	assert.Equal(t, "id1", attachments[0].Actions[0].Integration.Context["id"])
	assert.Equal(t, expectedHeader2, attachments[1].Text)
	assert.Equal(t, "id2", attachments[1].Actions[0].Integration.Context["id"])
}

func TestLoadMessages_ListIDsError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, store: mockStore}
	userID := "user1"
	expectedErr := errors.New("list error")

	mockStore.EXPECT().ListUserMessageIDs(userID).Return(nil, expectedErr)

	msgs, err := service.loadMessages(userID)

	assert.Nil(t, msgs)
	assert.Equal(t, expectedErr, err)
}

func TestLoadMessages_NoIDs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, store: mockStore}
	userID := "user1"

	mockStore.EXPECT().ListUserMessageIDs(userID).Return([]string{}, nil)

	msgs, err := service.loadMessages(userID)

	assert.NotNil(t, msgs)
	assert.Empty(t, msgs)
	assert.NoError(t, err)
}

func TestLoadMessages_GetMessageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, store: mockStore}
	userID := "user1"
	msgID := "id1"
	expectedErr := errors.New("get error")

	mockStore.EXPECT().ListUserMessageIDs(userID).Return([]string{msgID}, nil)
	mockStore.EXPECT().GetScheduledMessage(msgID).Return(nil, expectedErr)

	msgs, err := service.loadMessages(userID)

	assert.NotNil(t, msgs)
	assert.Empty(t, msgs)
	assert.NoError(t, err)
}

func TestLoadMessages_GetMessageNil_CleanupSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, store: mockStore}
	userID := "user1"
	msgID := "id1"

	mockStore.EXPECT().ListUserMessageIDs(userID).Return([]string{msgID}, nil)
	mockStore.EXPECT().GetScheduledMessage(msgID).Return(nil, nil)
	mockStore.EXPECT().CleanupMessageFromUserIndex(userID, msgID).Return(nil)

	msgs, err := service.loadMessages(userID)

	assert.NotNil(t, msgs)
	assert.Empty(t, msgs)
	assert.NoError(t, err)
}

func TestLoadMessages_GetMessageEmptyID_CleanupSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, store: mockStore}
	userID := "user1"
	msgID := "id1"
	emptyMsg := &types.ScheduledMessage{ID: ""}

	mockStore.EXPECT().ListUserMessageIDs(userID).Return([]string{msgID}, nil)
	mockStore.EXPECT().GetScheduledMessage(msgID).Return(emptyMsg, nil)
	mockStore.EXPECT().CleanupMessageFromUserIndex(userID, msgID).Return(nil)

	msgs, err := service.loadMessages(userID)

	assert.NotNil(t, msgs)
	assert.Empty(t, msgs)
	assert.NoError(t, err)
}

func TestLoadMessages_GetMessageNil_CleanupError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, store: mockStore}
	userID := "user1"
	msgID := "id1"
	cleanupErr := errors.New("cleanup failed")

	mockStore.EXPECT().ListUserMessageIDs(userID).Return([]string{msgID}, nil)
	mockStore.EXPECT().GetScheduledMessage(msgID).Return(nil, nil)
	mockStore.EXPECT().CleanupMessageFromUserIndex(userID, msgID).Return(cleanupErr)

	msgs, err := service.loadMessages(userID)

	assert.NotNil(t, msgs)
	assert.Empty(t, msgs)
	assert.NoError(t, err)
}

func TestLoadMessages_Success_SingleMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, store: mockStore}
	userID := "user1"
	msgID := "id1"
	expectedMsg := createTestMessage(msgID, userID, "ch1", "content", "UTC", time.Now())

	mockStore.EXPECT().ListUserMessageIDs(userID).Return([]string{msgID}, nil)
	mockStore.EXPECT().GetScheduledMessage(msgID).Return(expectedMsg, nil)

	msgs, err := service.loadMessages(userID)

	require.NoError(t, err)
	require.NotNil(t, msgs)
	require.Len(t, msgs, 1)
	assert.Equal(t, expectedMsg, msgs[0])
}

func TestLoadMessages_Success_MultipleMessages_Sorted(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, store: mockStore}
	userID := "user1"
	now := time.Now()
	msg1 := createTestMessage("id1", userID, "ch1", "content1", "UTC", now.Add(2*time.Hour))
	msg2 := createTestMessage("id2", userID, "ch1", "content2", "UTC", now.Add(1*time.Hour))
	msg3 := createTestMessage("id3", userID, "ch1", "content3", "UTC", now.Add(3*time.Hour))

	mockStore.EXPECT().ListUserMessageIDs(userID).Return([]string{"id1", "id2", "id3"}, nil)
	mockStore.EXPECT().GetScheduledMessage("id1").Return(msg1, nil)
	mockStore.EXPECT().GetScheduledMessage("id2").Return(msg2, nil)
	mockStore.EXPECT().GetScheduledMessage("id3").Return(msg3, nil)

	msgs, err := service.loadMessages(userID)

	require.NoError(t, err)
	require.NotNil(t, msgs)
	require.Len(t, msgs, 3)
	assert.Equal(t, msg2, msgs[0])
	assert.Equal(t, msg1, msgs[1])
	assert.Equal(t, msg3, msgs[2])
}

func TestLoadMessages_Success_MixedResults(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockStore := mock.NewMockStore(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, store: mockStore}
	userID := "user1"
	now := time.Now()

	msgOK1 := createTestMessage("id_ok1", userID, "ch1", "content1", "UTC", now.Add(2*time.Hour))
	msgOK2 := createTestMessage("id_ok2", userID, "ch1", "content2", "UTC", now.Add(1*time.Hour))
	idErr := "id_err"
	idNil := "id_nil"
	getErr := errors.New("get error")
	cleanupErr := errors.New("cleanup error")

	mockStore.EXPECT().ListUserMessageIDs(userID).Return([]string{"id_ok1", idErr, idNil, "id_ok2"}, nil)
	mockStore.EXPECT().GetScheduledMessage("id_ok1").Return(msgOK1, nil)
	mockStore.EXPECT().GetScheduledMessage(idErr).Return(nil, getErr)
	mockStore.EXPECT().GetScheduledMessage(idNil).Return(nil, nil)
	mockStore.EXPECT().CleanupMessageFromUserIndex(userID, idNil).Return(cleanupErr)
	mockStore.EXPECT().GetScheduledMessage("id_ok2").Return(msgOK2, nil)

	msgs, err := service.loadMessages(userID)

	require.NoError(t, err)
	require.NotNil(t, msgs)
	require.Len(t, msgs, 2)
	assert.Equal(t, msgOK2, msgs[0])
	assert.Equal(t, msgOK1, msgs[1])
}

func TestBuildAttachments_EmptyInput(t *testing.T) {
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger}

	attachments := service.buildAttachments([]*types.ScheduledMessage{})

	assert.NotNil(t, attachments)
	assert.Empty(t, attachments)
}

func TestBuildAttachments_SingleMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChannel := mock.NewMockChannelService(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, channel: mockChannel}

	now := time.Date(2023, 10, 27, 14, 30, 0, 0, time.UTC)
	msg := createTestMessage("msg1", "user1", "ch1", "Hello world", "UTC", now)
	info := &ports.ChannelInfo{ChannelID: "ch1", ChannelType: model.ChannelTypeOpen, ChannelLink: "~town-square"}
	channelLinkStr := "in channel: ~town-square"

	mockChannel.EXPECT().GetInfoOrUnknown("ch1").Return(info)
	mockChannel.EXPECT().MakeChannelLink(info).Return(channelLinkStr)

	attachments := service.buildAttachments([]*types.ScheduledMessage{msg})

	require.Len(t, attachments, 1)
	att := attachments[0]

	loc, _ := time.LoadLocation("UTC")
	expectedHeader := formatter.FormatListAttachmentHeader(now.In(loc), channelLinkStr, "Hello world")

	assert.Equal(t, expectedHeader, att.Text)
	require.Len(t, att.Actions, 1)
	action := att.Actions[0]
	assert.Equal(t, "delete", action.Id)
	assert.Equal(t, "Delete", action.Name)
	assert.Equal(t, "danger", action.Style)
	require.NotNil(t, action.Integration)
	assert.Equal(t, "/plugins/com.mattermost.plugin-poor-mans-scheduled-messages/api/v1/delete", action.Integration.URL)
	require.NotNil(t, action.Integration.Context)
	assert.Equal(t, "delete", action.Integration.Context["action"])
	assert.Equal(t, "msg1", action.Integration.Context["id"])
}

func TestBuildAttachments_MultipleMessages_SameChannel_CacheHit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChannel := mock.NewMockChannelService(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, channel: mockChannel}

	now := time.Now()
	msg1 := createTestMessage("msg1", "user1", "ch1", "content1", "UTC", now.Add(1*time.Hour))
	msg2 := createTestMessage("msg2", "user1", "ch1", "content2", "UTC", now.Add(2*time.Hour))
	info := &ports.ChannelInfo{ChannelID: "ch1", ChannelType: model.ChannelTypeOpen, ChannelLink: "~town-square"}
	channelLinkStr := "in channel: ~town-square"

	mockChannel.EXPECT().GetInfoOrUnknown("ch1").Return(info).Times(1)
	mockChannel.EXPECT().MakeChannelLink(info).Return(channelLinkStr).Times(2)

	attachments := service.buildAttachments([]*types.ScheduledMessage{msg1, msg2})

	require.Len(t, attachments, 2)
	assert.Contains(t, attachments[0].Text, "content1")
	assert.Equal(t, "msg1", attachments[0].Actions[0].Integration.Context["id"])
	assert.Contains(t, attachments[1].Text, "content2")
	assert.Equal(t, "msg2", attachments[1].Actions[0].Integration.Context["id"])
}

func TestBuildAttachments_MultipleMessages_DifferentChannels(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChannel := mock.NewMockChannelService(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, channel: mockChannel}

	now := time.Now()
	msg1 := createTestMessage("msg1", "user1", "ch1", "content1", "UTC", now.Add(1*time.Hour))
	msg2 := createTestMessage("msg2", "user1", "ch2", "content2", "UTC", now.Add(2*time.Hour))
	info1 := &ports.ChannelInfo{ChannelID: "ch1", ChannelType: model.ChannelTypeOpen, ChannelLink: "~town-square"}
	info2 := &ports.ChannelInfo{ChannelID: "ch2", ChannelType: model.ChannelTypeDirect, ChannelLink: "@otheruser"}
	linkStr1 := "in channel: ~town-square"
	linkStr2 := "in direct message with: @otheruser"

	mockChannel.EXPECT().GetInfoOrUnknown("ch1").Return(info1)
	mockChannel.EXPECT().GetInfoOrUnknown("ch2").Return(info2)
	mockChannel.EXPECT().MakeChannelLink(info1).Return(linkStr1)
	mockChannel.EXPECT().MakeChannelLink(info2).Return(linkStr2)

	attachments := service.buildAttachments([]*types.ScheduledMessage{msg1, msg2})

	require.Len(t, attachments, 2)
	assert.Contains(t, attachments[0].Text, linkStr1)
	assert.Contains(t, attachments[0].Text, "content1")
	assert.Equal(t, "msg1", attachments[0].Actions[0].Integration.Context["id"])
	assert.Contains(t, attachments[1].Text, linkStr2)
	assert.Contains(t, attachments[1].Text, "content2")
	assert.Equal(t, "msg2", attachments[1].Actions[0].Integration.Context["id"])
}

func TestBuildAttachments_TimezoneHandling(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockChannel := mock.NewMockChannelService(ctrl)
	logger := testutil.FakeLogger{}
	service := &ListService{logger: logger, channel: mockChannel}

	// 2 PM UTC is 10 AM EDT (UTC-4)
	postAtUTC := time.Date(2023, 7, 4, 14, 0, 0, 0, time.UTC)
	timezone := "America/New_York"
	msg := createTestMessage("msg1", "user1", "ch1", "Timezone test", timezone, postAtUTC)
	info := &ports.ChannelInfo{ChannelID: "ch1", ChannelType: model.ChannelTypeOpen, ChannelLink: "~test"}
	linkStr := "in channel: ~test"

	mockChannel.EXPECT().GetInfoOrUnknown("ch1").Return(info)
	mockChannel.EXPECT().MakeChannelLink(info).Return(linkStr)

	attachments := service.buildAttachments([]*types.ScheduledMessage{msg})

	require.Len(t, attachments, 1)
	att := attachments[0]

	// Expect time formatted in America/New_York (EDT on July 4th)
	locNY, err := time.LoadLocation(timezone)
	require.NoError(t, err)
	expectedTimeStr := postAtUTC.In(locNY).Format(constants.TimeLayout) // Should be 10:00 AM

	expectedHeader := formatter.FormatListAttachmentHeader(postAtUTC.In(locNY), linkStr, "Timezone test")
	assert.Equal(t, expectedHeader, att.Text)
	assert.Contains(t, att.Text, expectedTimeStr)
	assert.Contains(t, att.Text, "10:00 AM")
}

func TestErrorResponse(t *testing.T) {
	text := "This is an error"
	resp := errorResponse(text)

	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	assert.Equal(t, text, resp.Text)
	assert.Empty(t, resp.Props)
}

func TestEmptyResponse(t *testing.T) {
	resp := emptyResponse()

	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	assert.Equal(t, constants.EmptyListMessage, resp.Text)
	assert.Empty(t, resp.Props)
}

func TestSuccessResponse(t *testing.T) {
	att1 := &model.SlackAttachment{Text: "att1"}
	att2 := &model.SlackAttachment{Text: "att2"}
	atts := []*model.SlackAttachment{att1, att2}

	resp := successResponse(atts)

	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	assert.Equal(t, constants.ListHeader, resp.Text)
	require.NotNil(t, resp.Props)
	respAtts, ok := resp.Props["attachments"].([]*model.SlackAttachment)
	require.True(t, ok)
	assert.Equal(t, atts, respAtts)
}

func TestCreateAttachment(t *testing.T) {
	text := "Attachment header text"
	messageID := "msg-abc-123"

	att := createAttachment(text, messageID)

	assert.Equal(t, text, att.Text)
	require.Len(t, att.Actions, 1)
	action := att.Actions[0]
	assert.Equal(t, "delete", action.Id)
	assert.Equal(t, "Delete", action.Name)
	assert.Equal(t, "danger", action.Style)
	require.NotNil(t, action.Integration)
	assert.Equal(t, "/plugins/com.mattermost.plugin-poor-mans-scheduled-messages/api/v1/delete", action.Integration.URL)
	require.NotNil(t, action.Integration.Context)
	assert.Equal(t, "delete", action.Integration.Context["action"])
	assert.Equal(t, messageID, action.Integration.Context["id"])
}
