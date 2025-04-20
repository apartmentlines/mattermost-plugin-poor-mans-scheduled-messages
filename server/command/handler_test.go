package command_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/command"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testMocks struct {
	logger          *testutil.FakeLogger
	slasher         *mock.MockSlashCommandService
	user            *mock.MockUserService
	store           *mock.MockStore
	channel         *mock.MockChannelService
	listService     *mock.MockListService
	scheduleService *mock.MockScheduleService
}

func setup(t *testing.T) (*command.Handler, *testMocks, *gomock.Controller) {
	t.Helper()
	ctrl := gomock.NewController(t)
	mocks := &testMocks{
		logger:          &testutil.FakeLogger{},
		slasher:         mock.NewMockSlashCommandService(ctrl),
		user:            mock.NewMockUserService(ctrl),
		store:           mock.NewMockStore(ctrl),
		channel:         mock.NewMockChannelService(ctrl),
		listService:     mock.NewMockListService(ctrl),
		scheduleService: mock.NewMockScheduleService(ctrl),
	}

	helpText := "Sample help text"

	handler := command.NewHandler(
		mocks.logger,
		mocks.slasher,
		mocks.user,
		mocks.store,
		mocks.channel,
		mocks.listService,
		mocks.scheduleService,
		helpText,
	)
	require.NotNil(t, handler)
	return handler, mocks, ctrl
}

func TestNewHandler_SuccessfulInstantiation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockSlasher := mock.NewMockSlashCommandService(ctrl)
	mockUser := mock.NewMockUserService(ctrl)
	mockStore := mock.NewMockStore(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)
	mockListService := mock.NewMockListService(ctrl)
	mockScheduleService := mock.NewMockScheduleService(ctrl)
	helpText := "Test Help"

	handler := command.NewHandler(
		&testutil.FakeLogger{},
		mockSlasher,
		mockUser,
		mockStore,
		mockChannel,
		mockListService,
		mockScheduleService,
		helpText,
	)

	assert.NotNil(t, handler)
}

func TestRegister_SuccessfulRegistration(t *testing.T) {
	handler, mocks, ctrl := setup(t)
	defer ctrl.Finish()

	mocks.slasher.EXPECT().Register(gomock.Any()).Return(nil)

	err := handler.Register()

	assert.NoError(t, err)
}

func TestRegister_RegistrationFailure(t *testing.T) {
	handler, mocks, ctrl := setup(t)
	defer ctrl.Finish()

	testErr := errors.New("registration failed")

	mocks.slasher.EXPECT().Register(gomock.Any()).Return(testErr)

	err := handler.Register()

	assert.EqualError(t, err, testErr.Error())
}

func TestExecute_HelpSubcommand(t *testing.T) {
	handler, _, ctrl := setup(t)
	defer ctrl.Finish()

	helpText := "Sample help text"
	args := &model.CommandArgs{
		UserId:    "testUserID",
		ChannelId: "testChannelID",
		Command:   "/" + constants.CommandTrigger + " " + constants.SubcommandHelp,
	}

	resp, appErr := handler.Execute(args)

	require.Nil(t, appErr)
	require.NotNil(t, resp)
	assert.Equal(t, model.CommandResponseTypeEphemeral, resp.ResponseType)
	assert.Equal(t, helpText, resp.Text)
}

func TestExecute_ListSubcommand(t *testing.T) {
	handler, mocks, ctrl := setup(t)
	defer ctrl.Finish()

	userID := "testUserID"
	args := &model.CommandArgs{
		UserId:    userID,
		ChannelId: "testChannelID",
		Command:   "/" + constants.CommandTrigger + " " + constants.SubcommandList,
	}
	expectedResp := &model.CommandResponse{Text: "List response"}

	mocks.listService.EXPECT().Build(userID).Return(expectedResp)

	resp, appErr := handler.Execute(args)

	require.Nil(t, appErr)
	assert.Equal(t, expectedResp, resp)
}

func TestExecute_ScheduleSubcommand_Default(t *testing.T) {
	handler, mocks, ctrl := setup(t)
	defer ctrl.Finish()

	userID := "testUserID"
	commandText := "at 10am message test"
	args := &model.CommandArgs{
		UserId:    userID,
		ChannelId: "testChannelID",
		Command:   "/" + constants.CommandTrigger + " " + commandText,
	}
	expectedResp := &model.CommandResponse{Text: "Schedule response"}

	mocks.scheduleService.EXPECT().Build(args, commandText).Return(expectedResp)

	resp, appErr := handler.Execute(args)

	require.Nil(t, appErr)
	assert.Equal(t, expectedResp, resp)
}

func TestExecute_ScheduleSubcommand_Empty(t *testing.T) {
	handler, mocks, ctrl := setup(t)
	defer ctrl.Finish()

	userID := "testUserID"
	args := &model.CommandArgs{
		UserId:    userID,
		ChannelId: "testChannelID",
		Command:   "/" + constants.CommandTrigger + "  ",
	}
	expectedResp := &model.CommandResponse{Text: "Empty schedule response"}

	mocks.scheduleService.EXPECT().Build(args, "").Return(expectedResp)

	resp, appErr := handler.Execute(args)

	require.Nil(t, appErr)
	assert.Equal(t, expectedResp, resp)
}

func TestBuildEphemeralList_SuccessfulListBuild(t *testing.T) {
	handler, mocks, ctrl := setup(t)
	defer ctrl.Finish()

	userID := "testUserID"
	args := &model.CommandArgs{UserId: userID}
	expectedResp := &model.CommandResponse{Text: "List built"}

	mocks.listService.EXPECT().Build(userID).Return(expectedResp)

	resp := handler.BuildEphemeralList(args)

	assert.Equal(t, expectedResp, resp)
}

func TestUserDeleteMessage_SuccessfulDeletion(t *testing.T) {
	handler, mocks, ctrl := setup(t)
	defer ctrl.Finish()

	userID := "ownerUserID"
	msgID := "testMsgID"
	msg := &types.ScheduledMessage{ID: msgID, UserID: userID}

	mocks.store.EXPECT().GetScheduledMessage(msgID).Return(msg, nil)
	mocks.store.EXPECT().DeleteScheduledMessage(userID, msgID).Return(nil)

	returnedMsg, err := handler.UserDeleteMessage(userID, msgID)

	require.NoError(t, err)
	assert.Equal(t, msg, returnedMsg)
}

func TestUserDeleteMessage_Failure_GetScheduledMessageFails(t *testing.T) {
	handler, mocks, ctrl := setup(t)
	defer ctrl.Finish()

	userID := "testUserID"
	msgID := "testMsgID"
	getErr := errors.New("get failed")
	expectedErr := fmt.Errorf("failed to get scheduled message %s: %w", msgID, getErr)

	mocks.store.EXPECT().GetScheduledMessage(msgID).Return(nil, getErr)

	returnedMsg, err := handler.UserDeleteMessage(userID, msgID)

	require.Error(t, err)
	assert.Nil(t, returnedMsg)
	assert.EqualError(t, err, expectedErr.Error())
}

func TestUserDeleteMessage_Failure_OwnershipMismatch(t *testing.T) {
	handler, mocks, ctrl := setup(t)
	defer ctrl.Finish()

	requestingUserID := "requesterID"
	ownerUserID := "ownerID"
	msgID := "testMsgID"
	msg := &types.ScheduledMessage{ID: msgID, UserID: ownerUserID}
	expectedErr := fmt.Errorf("user %s attempted to delete message %s owned by %s", requestingUserID, msgID, ownerUserID)

	mocks.store.EXPECT().GetScheduledMessage(msgID).Return(msg, nil)

	returnedMsg, err := handler.UserDeleteMessage(requestingUserID, msgID)

	require.Error(t, err)
	assert.Nil(t, returnedMsg)
	assert.EqualError(t, err, expectedErr.Error())
}

func TestUserDeleteMessage_Failure_DeleteScheduledMessageFails(t *testing.T) {
	handler, mocks, ctrl := setup(t)
	defer ctrl.Finish()

	userID := "ownerUserID"
	msgID := "testMsgID"
	msg := &types.ScheduledMessage{ID: msgID, UserID: userID}
	deleteErr := errors.New("delete failed")
	expectedErr := fmt.Errorf("failed to delete scheduled message %s: %w", msgID, deleteErr)

	mocks.store.EXPECT().GetScheduledMessage(msgID).Return(msg, nil)
	mocks.store.EXPECT().DeleteScheduledMessage(userID, msgID).Return(deleteErr)

	returnedMsg, err := handler.UserDeleteMessage(userID, msgID)

	require.Error(t, err)
	assert.Nil(t, returnedMsg)
	assert.EqualError(t, err, expectedErr.Error())
}
