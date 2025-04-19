package command

import (
	"errors"
	"fmt"
	"strings"
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

const (
	testUserID        = "test-user-id"
	testChannelID     = "test-channel-id"
	testMsgID         = "test-msg-id"
	testMaxUserMsgs   = 5
	testTimezone      = "America/New_York"
	testDefaultTZ     = "UTC"
	testChannelLink   = "~town-square"
	testTeamName      = "Test Team"
	testFormattedLink = "in channel: ~town-square"
)

var testNow = time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

type testMocks struct {
	ctrl    *gomock.Controller
	userAPI *mock.MockUserService
	store   *mock.MockStore
	channel *mock.MockChannelService
	clock   *testutil.FakeClock
	logger  *testutil.FakeLogger
}

func setupScheduleServiceTest(t *testing.T) (*ScheduleService, *testMocks) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mocks := &testMocks{
		ctrl:    ctrl,
		userAPI: mock.NewMockUserService(ctrl),
		store:   mock.NewMockStore(ctrl),
		channel: mock.NewMockChannelService(ctrl),
		clock:   &testutil.FakeClock{NowTime: testNow},
		logger:  &testutil.FakeLogger{},
	}

	service := NewScheduleService(
		mocks.logger,
		mocks.userAPI,
		mocks.store,
		mocks.channel,
		mocks.clock,
		testMaxUserMsgs,
	)
	require.NotNil(t, service)
	return service, mocks
}

func defaultArgs() *model.CommandArgs {
	return &model.CommandArgs{
		UserId:    testUserID,
		ChannelId: testChannelID,
	}
}

func TestNewScheduleService(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)

	assert.Equal(t, mocks.logger, service.logger)
	assert.Equal(t, mocks.userAPI, service.userAPI)
	assert.Equal(t, mocks.store, service.store)
	assert.Equal(t, mocks.channel, service.channel)
	assert.Equal(t, mocks.clock, service.clock)
	assert.Equal(t, testMaxUserMsgs, service.maxUserMessages)
}

func TestBuild_HappyPath(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	text := "at 3:00PM on 2024-01-16 message Hello world"
	expectedPostAtUTC := time.Date(2024, 1, 16, 20, 0, 0, 0, time.UTC) // 3 PM EST is 8 PM UTC
	expectedPostAtLocal := time.Date(2024, 1, 16, 15, 0, 0, 0, testutil.MustLoadLocation(t, testTimezone))
	channelInfo := &ports.ChannelInfo{ChannelID: testChannelID, ChannelLink: testChannelLink, TeamName: testTeamName, ChannelType: model.ChannelTypeOpen}

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return([]string{"id1"}, nil)
	mocks.userAPI.EXPECT().Get(testUserID).Return(&model.User{Timezone: map[string]string{"manualTimezone": testTimezone}}, nil)
	mocks.store.EXPECT().GenerateMessageID().Return(testMsgID)
	mocks.store.EXPECT().SaveScheduledMessage(testUserID, gomock.AssignableToTypeOf(&types.ScheduledMessage{})).
		DoAndReturn(func(userID string, msg *types.ScheduledMessage) error {
			assert.Equal(t, testMsgID, msg.ID)
			assert.Equal(t, testUserID, msg.UserID)
			assert.Equal(t, testChannelID, msg.ChannelID)
			assert.True(t, expectedPostAtUTC.Equal(msg.PostAt), "Expected %v, got %v", expectedPostAtUTC, msg.PostAt)
			assert.Equal(t, "Hello world", msg.MessageContent)
			assert.Equal(t, testTimezone, msg.Timezone)
			return nil
		})
	mocks.channel.EXPECT().GetInfoOrUnknown(testChannelID).Return(channelInfo)
	mocks.channel.EXPECT().MakeChannelLink(channelInfo).Return(testFormattedLink)

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	expectedSuccessMsg := formatter.FormatScheduleSuccess(expectedPostAtLocal, testTimezone, testFormattedLink)
	assert.Equal(t, expectedSuccessMsg, resp.Text)
}

func TestBuild_TimezoneLogic_DefaultUsed_NoSettings(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	text := "at 3:00PM on 2024-01-16 message Hello UTC"
	// 3 PM UTC on Jan 16
	expectedPostAtUTC := time.Date(2024, 1, 16, 15, 0, 0, 0, time.UTC)
	expectedPostAtLocal := expectedPostAtUTC
	channelInfo := &ports.ChannelInfo{ChannelID: testChannelID, ChannelLink: testChannelLink, TeamName: testTeamName, ChannelType: model.ChannelTypeOpen}

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return([]string{}, nil)
	mocks.userAPI.EXPECT().Get(testUserID).Return(&model.User{Timezone: map[string]string{}}, nil)
	mocks.store.EXPECT().GenerateMessageID().Return(testMsgID)
	mocks.store.EXPECT().SaveScheduledMessage(testUserID, gomock.AssignableToTypeOf(&types.ScheduledMessage{})).
		DoAndReturn(func(userID string, msg *types.ScheduledMessage) error {
			assert.Equal(t, testDefaultTZ, msg.Timezone)
			assert.True(t, expectedPostAtUTC.Equal(msg.PostAt))
			return nil
		})
	mocks.channel.EXPECT().GetInfoOrUnknown(testChannelID).Return(channelInfo)
	mocks.channel.EXPECT().MakeChannelLink(channelInfo).Return(testFormattedLink)

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	expectedSuccessMsg := formatter.FormatScheduleSuccess(expectedPostAtLocal, testDefaultTZ, testFormattedLink)
	assert.Equal(t, expectedSuccessMsg, resp.Text)
}

func TestBuild_TimezoneLogic_ManualUsed(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	autoTZ := "America/Los_Angeles"
	manualTZ := "America/New_York"
	text := "at 3:00PM on 2024-01-16 message Hello EST"
	// 3 PM EST on Jan 16 is 8 PM UTC on Jan 16
	expectedPostAtUTC := time.Date(2024, 1, 16, 20, 0, 0, 0, time.UTC)
	expectedPostAtLocal := time.Date(2024, 1, 16, 15, 0, 0, 0, testutil.MustLoadLocation(t, manualTZ))
	channelInfo := &ports.ChannelInfo{ChannelID: testChannelID, ChannelLink: testChannelLink, TeamName: testTeamName, ChannelType: model.ChannelTypeOpen}

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return([]string{}, nil)
	mocks.userAPI.EXPECT().Get(testUserID).Return(&model.User{Timezone: map[string]string{
		"useAutomaticTimezone": "false", // Important
		"automaticTimezone":    autoTZ,
		"manualTimezone":       manualTZ,
	}}, nil)
	mocks.store.EXPECT().GenerateMessageID().Return(testMsgID)
	mocks.store.EXPECT().SaveScheduledMessage(testUserID, gomock.AssignableToTypeOf(&types.ScheduledMessage{})).
		DoAndReturn(func(userID string, msg *types.ScheduledMessage) error {
			assert.Equal(t, manualTZ, msg.Timezone)
			assert.True(t, expectedPostAtUTC.Equal(msg.PostAt))
			return nil
		})
	mocks.channel.EXPECT().GetInfoOrUnknown(testChannelID).Return(channelInfo)
	mocks.channel.EXPECT().MakeChannelLink(channelInfo).Return(testFormattedLink)

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	expectedSuccessMsg := formatter.FormatScheduleSuccess(expectedPostAtLocal, manualTZ, testFormattedLink)
	assert.Equal(t, expectedSuccessMsg, resp.Text)
}

func TestBuild_PreparationFailure_TimeResolutionError(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	text := "at 9:00AM on 2024-01-15 message Hello Past" // 9 AM UTC on Jan 15, testNow is 10 AM UTC Jan 15

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return([]string{}, nil)
	mocks.userAPI.EXPECT().Get(testUserID).Return(&model.User{Timezone: map[string]string{"manualTimezone": testDefaultTZ}}, nil) // Use UTC

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	assert.Contains(t, resp.Text, "Error preparing schedule:")
	assert.Contains(t, resp.Text, "failed to resolve time:")
	assert.Contains(t, resp.Text, "is already in the past")
	assert.Contains(t, resp.Text, "Original input: `at 9:00AM on 2024-01-15 message Hello Past`")
}

func TestBuild_PersistenceFailure_StoreSaveError(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	text := "at 3:00PM on 2024-01-16 message Hello world"
	expectedPostAtLocal := time.Date(2024, 1, 16, 15, 0, 0, 0, testutil.MustLoadLocation(t, testTimezone))
	channelInfo := &ports.ChannelInfo{ChannelID: testChannelID, ChannelLink: testChannelLink, TeamName: testTeamName, ChannelType: model.ChannelTypeOpen}
	saveErr := errors.New("kv set failed")

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return([]string{}, nil)
	mocks.userAPI.EXPECT().Get(testUserID).Return(&model.User{Timezone: map[string]string{"manualTimezone": testTimezone}}, nil)
	mocks.store.EXPECT().GenerateMessageID().Return(testMsgID)
	mocks.store.EXPECT().SaveScheduledMessage(testUserID, gomock.AssignableToTypeOf(&types.ScheduledMessage{})).Return(saveErr)
	mocks.channel.EXPECT().GetInfoOrUnknown(testChannelID).Return(channelInfo)
	mocks.channel.EXPECT().MakeChannelLink(channelInfo).Return(testFormattedLink)

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	expectedFormattedErr := formatter.FormatScheduleError(expectedPostAtLocal, testTimezone, testFormattedLink, saveErr)
	assert.Equal(t, expectedFormattedErr, resp.Text)
}

func TestBuild_TimezoneLogic_AutomaticUsed(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	autoTZ := "America/Los_Angeles"
	manualTZ := "America/New_York"
	text := "at 3:00PM on 2024-01-16 message Hello PST"
	// 3 PM PST on Jan 16 is 11 PM UTC on Jan 16
	expectedPostAtUTC := time.Date(2024, 1, 16, 23, 0, 0, 0, time.UTC)
	expectedPostAtLocal := time.Date(2024, 1, 16, 15, 0, 0, 0, testutil.MustLoadLocation(t, autoTZ))
	channelInfo := &ports.ChannelInfo{ChannelID: testChannelID, ChannelLink: testChannelLink, TeamName: testTeamName, ChannelType: model.ChannelTypeOpen}

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return([]string{}, nil)
	mocks.userAPI.EXPECT().Get(testUserID).Return(&model.User{Timezone: map[string]string{
		"useAutomaticTimezone": "true",
		"automaticTimezone":    autoTZ,
		"manualTimezone":       manualTZ,
	}}, nil)
	mocks.store.EXPECT().GenerateMessageID().Return(testMsgID)
	mocks.store.EXPECT().SaveScheduledMessage(testUserID, gomock.AssignableToTypeOf(&types.ScheduledMessage{})).
		DoAndReturn(func(userID string, msg *types.ScheduledMessage) error {
			assert.Equal(t, autoTZ, msg.Timezone)
			assert.True(t, expectedPostAtUTC.Equal(msg.PostAt))
			return nil
		})
	mocks.channel.EXPECT().GetInfoOrUnknown(testChannelID).Return(channelInfo)
	mocks.channel.EXPECT().MakeChannelLink(channelInfo).Return(testFormattedLink)

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	expectedSuccessMsg := formatter.FormatScheduleSuccess(expectedPostAtLocal, autoTZ, testFormattedLink)
	assert.Equal(t, expectedSuccessMsg, resp.Text)
}

func TestBuild_ValidationFailure_MaxUserMessagesReached(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	text := "at 3:00PM message Hello"
	existingIDs := make([]string, testMaxUserMsgs)
	for i := range testMaxUserMsgs {
		existingIDs[i] = fmt.Sprintf("id%d", i)
	}

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return(existingIDs, nil)

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	expectedErr := fmt.Errorf("cannot schedule more than %d messages (current: %d)", testMaxUserMsgs, testMaxUserMsgs)
	expectedFormattedErr := formatter.FormatScheduleValidationError(expectedErr)
	assert.Equal(t, expectedFormattedErr, resp.Text)
}

func TestBuild_PreparationFailure_InputParsingError(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	text := "at noon tomorrow do stuff"

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return([]string{}, nil)

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	assert.Contains(t, resp.Text, "Error preparing schedule:")
	assert.Contains(t, resp.Text, constants.ParserErrInvalidFormat)
	assert.Contains(t, resp.Text, "Original input: `at noon tomorrow do stuff`")
}

func TestBuild_PreparationFailure_UserTimezoneFetchError(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	text := "at 3:00PM on 2024-01-16 message Hello world"
	expectedPostAtUTC := time.Date(2024, 1, 16, 15, 0, 0, 0, time.UTC)
	expectedPostAtLocal := expectedPostAtUTC
	channelInfo := &ports.ChannelInfo{ChannelID: testChannelID, ChannelLink: testChannelLink, TeamName: testTeamName, ChannelType: model.ChannelTypeOpen}
	fetchErr := errors.New("api error")

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return([]string{}, nil)
	mocks.userAPI.EXPECT().Get(testUserID).Return(nil, fetchErr)
	mocks.store.EXPECT().GenerateMessageID().Return(testMsgID)
	mocks.store.EXPECT().SaveScheduledMessage(testUserID, gomock.AssignableToTypeOf(&types.ScheduledMessage{})).
		DoAndReturn(func(userID string, msg *types.ScheduledMessage) error {
			assert.Equal(t, testMsgID, msg.ID)
			assert.True(t, expectedPostAtUTC.Equal(msg.PostAt), "Expected %v, got %v", expectedPostAtUTC, msg.PostAt)
			assert.Equal(t, testDefaultTZ, msg.Timezone)
			return nil
		})
	mocks.channel.EXPECT().GetInfoOrUnknown(testChannelID).Return(channelInfo)
	mocks.channel.EXPECT().MakeChannelLink(channelInfo).Return(testFormattedLink)

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	expectedSuccessMsg := formatter.FormatScheduleSuccess(expectedPostAtLocal, testDefaultTZ, testFormattedLink)
	assert.Equal(t, expectedSuccessMsg, resp.Text) // Response should show UTC
}

func TestBuild_PreparationFailure_InvalidUserTimezone(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	invalidTZ := "Invalid/Timezone"
	text := "at 3:00PM on 2024-01-16 message Hello world"
	expectedPostAtUTC := time.Date(2024, 1, 16, 15, 0, 0, 0, time.UTC)
	expectedPostAtLocal := expectedPostAtUTC
	channelInfo := &ports.ChannelInfo{ChannelID: testChannelID, ChannelLink: testChannelLink, TeamName: testTeamName, ChannelType: model.ChannelTypeOpen}

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return([]string{}, nil)
	mocks.userAPI.EXPECT().Get(testUserID).Return(&model.User{Timezone: map[string]string{"manualTimezone": invalidTZ}}, nil)
	mocks.store.EXPECT().GenerateMessageID().Return(testMsgID)
	mocks.store.EXPECT().SaveScheduledMessage(testUserID, gomock.AssignableToTypeOf(&types.ScheduledMessage{})).
		DoAndReturn(func(userID string, msg *types.ScheduledMessage) error {
			assert.Equal(t, testMsgID, msg.ID)
			assert.True(t, expectedPostAtUTC.Equal(msg.PostAt), "Expected %v, got %v", expectedPostAtUTC, msg.PostAt)
			assert.Equal(t, constants.DefaultTimezone, msg.Timezone)
			return nil
		})
	mocks.channel.EXPECT().GetInfoOrUnknown(testChannelID).Return(channelInfo)
	mocks.channel.EXPECT().MakeChannelLink(channelInfo).Return(testFormattedLink)

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	expectedSuccessMsg := formatter.FormatScheduleSuccess(expectedPostAtLocal, testDefaultTZ, testFormattedLink) // Show UTC in response
	assert.Equal(t, expectedSuccessMsg, resp.Text)
}

func TestBuild_ValidationFailure_EmptyCommandText(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	text := "   "

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return([]string{}, nil)

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	expectedFormattedErr := formatter.FormatEmptyCommandError()
	assert.Equal(t, expectedFormattedErr, resp.Text)
}

func TestBuild_ValidationFailure_MaxMessageBytesExceeded(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	longMessage := strings.Repeat("a", constants.MaxMessageBytes+1)
	text := fmt.Sprintf("at 3:00PM message %s", longMessage)

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return([]string{}, nil)

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	kb := float64(constants.MaxMessageBytes) / 1024
	userKb := float64(len(text)) / 1024
	expectedErr := fmt.Errorf("message length %.2f KB exceeds limit %.2f KB", userKb, kb)
	expectedFormattedErr := formatter.FormatScheduleValidationError(expectedErr)
	assert.Equal(t, expectedFormattedErr, resp.Text)
}

func TestBuild_ValidationFailure_ErrorCheckingMaxUserMessages(t *testing.T) {
	service, mocks := setupScheduleServiceTest(t)
	args := defaultArgs()
	text := "at 3:00PM message Hello"
	storeErr := errors.New("kv error")

	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return(nil, storeErr)

	resp := service.Build(args, text)

	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	expectedErr := fmt.Errorf("failed to check message count: %w", storeErr)
	expectedFormattedErr := formatter.FormatScheduleValidationError(expectedErr)
	assert.Equal(t, expectedFormattedErr, resp.Text)
}
