package scheduler

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mm"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost/server/public/model"
)

func TestProcessDueMessages_PostSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPoster := mock.NewMockPostService(ctrl)
	mockKV := mock.NewMockKVService(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)

	st := store.NewKVStore(testutil.FakeLogger{}, mockKV, mm.ListMatchingService{}, constants.MaxUserMessages)
	clk := testutil.FakeClock{NowTime: time.Now().UTC()}
	s := New(testutil.FakeLogger{}, mockPoster, st, mockChannel, "bot", clk)

	now := clk.Now()
	msg := &types.ScheduledMessage{
		ID:             "uuid-1",
		UserID:         "user",
		ChannelID:      "chan",
		PostAt:         now.Add(-time.Minute),
		MessageContent: "hi",
		Timezone:       "UTC",
	}
	msgKey := testutil.SchedKey(msg.ID)
	userIndexKey := testutil.IndexKey(msg.UserID)

	mockKV.EXPECT().ListKeys(0, constants.MaxUserMessages, gomock.Any()).Return([]string{msgKey}, nil)
	mockKV.EXPECT().Get(msgKey, gomock.Any()).SetArg(1, *msg).Return(nil).Times(2)
	mockKV.EXPECT().Get(userIndexKey, gomock.Any()).SetArg(1, []string{msg.ID}).Return(nil)
	mockKV.EXPECT().Set(userIndexKey, gomock.Eq([]string{})).Return(true, nil)
	mockKV.EXPECT().Delete(msgKey).Return(nil)
	mockPoster.EXPECT().CreatePost(gomock.Eq(&model.Post{
		ChannelId: msg.ChannelID,
		Message:   msg.MessageContent,
		UserId:    msg.UserID,
	})).Return(nil)

	s.processDueMessages()
}

func TestProcessDueMessages_PostFailure(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPoster := mock.NewMockPostService(ctrl)
	mockKV := mock.NewMockKVService(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)

	st := store.NewKVStore(testutil.FakeLogger{}, mockKV, mm.ListMatchingService{}, constants.MaxUserMessages)
	clk := testutil.FakeClock{NowTime: time.Now().UTC()}
	s := New(testutil.FakeLogger{}, mockPoster, st, mockChannel, "bot", clk)

	now := clk.Now()
	msg := &types.ScheduledMessage{
		ID:             "uuid-2",
		UserID:         "user",
		ChannelID:      "chan",
		PostAt:         now.Add(-time.Minute),
		MessageContent: "hi",
		Timezone:       "UTC",
	}
	msgKey := testutil.SchedKey(msg.ID)
	userIndexKey := testutil.IndexKey(msg.UserID)
	postErr := errors.New("fail")
	channelInfo := &ports.ChannelInfo{ChannelID: msg.ChannelID, ChannelLink: "some-link"}

	mockKV.EXPECT().ListKeys(0, constants.MaxUserMessages, gomock.Any()).Return([]string{msgKey}, nil)
	mockKV.EXPECT().Get(msgKey, gomock.Any()).SetArg(1, *msg).Return(nil).Times(2)
	mockKV.EXPECT().Get(userIndexKey, gomock.Any()).SetArg(1, []string{msg.ID}).Return(nil)
	mockKV.EXPECT().Set(userIndexKey, gomock.Eq([]string{})).Return(true, nil)
	mockKV.EXPECT().Delete(msgKey).Return(nil)
	mockPoster.EXPECT().CreatePost(gomock.Any()).Return(postErr)
	mockChannel.EXPECT().GetInfoOrUnknown(msg.ChannelID).Return(channelInfo)
	mockChannel.EXPECT().MakeChannelLink(channelInfo).Return("in channel: some-link")
	mockPoster.EXPECT().DM("bot", msg.UserID, gomock.Any()).Return(nil)

	s.processDueMessages()
}

func TestProcessDueMessages_NotDueYet(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPoster := mock.NewMockPostService(ctrl)
	mockKV := mock.NewMockKVService(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)

	st := store.NewKVStore(testutil.FakeLogger{}, mockKV, mm.ListMatchingService{}, constants.MaxUserMessages)
	clk := testutil.FakeClock{NowTime: time.Now().UTC()}
	s := New(testutil.FakeLogger{}, mockPoster, st, mockChannel, "bot", clk)

	now := clk.Now()
	msg := &types.ScheduledMessage{
		ID:             "uuid-3",
		UserID:         "user",
		ChannelID:      "chan",
		PostAt:         now.Add(time.Minute),
		MessageContent: "hi",
		Timezone:       "UTC",
	}
	msgKey := testutil.SchedKey(msg.ID)

	mockKV.EXPECT().ListKeys(0, constants.MaxUserMessages, gomock.Any()).Return([]string{msgKey}, nil)
	mockKV.EXPECT().Get(msgKey, gomock.Any()).SetArg(1, *msg).Return(nil)

	s.processDueMessages()
}

func TestProcessDueMessages_ListError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPoster := mock.NewMockPostService(ctrl)
	mockKV := mock.NewMockKVService(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)

	st := store.NewKVStore(testutil.FakeLogger{}, mockKV, mm.ListMatchingService{}, constants.MaxUserMessages)
	clk := testutil.FakeClock{NowTime: time.Now().UTC()}
	s := New(testutil.FakeLogger{}, mockPoster, st, mockChannel, "bot", clk)

	mockKV.EXPECT().ListKeys(0, constants.MaxUserMessages, gomock.Any()).Return(nil, errors.New("boom"))

	s.processDueMessages()
}

func TestScheduler_StartAndStop(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPoster := mock.NewMockPostService(ctrl)
	mockKV := mock.NewMockKVService(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)

	st := store.NewKVStore(testutil.FakeLogger{}, mockKV, mm.ListMatchingService{}, constants.MaxUserMessages)
	clk := testutil.FakeClock{NowTime: time.Now().UTC()}
	s := New(testutil.FakeLogger{}, mockPoster, st, mockChannel, "bot", clk)
	s.newTicker = func(d time.Duration) *time.Ticker { return time.NewTicker(1 * time.Millisecond) }

	now := clk.Now()
	msg := &types.ScheduledMessage{
		ID:             "uuid-4",
		UserID:         "user",
		ChannelID:      "chan",
		PostAt:         now.Add(-time.Minute),
		MessageContent: "hi",
		Timezone:       "UTC",
	}
	msgKey := testutil.SchedKey(msg.ID)
	userIndexKey := testutil.IndexKey(msg.UserID)

	mockKV.EXPECT().ListKeys(0, constants.MaxUserMessages, gomock.Any()).Return([]string{msgKey}, nil).MinTimes(1)
	mockKV.EXPECT().Get(msgKey, gomock.Any()).SetArg(1, *msg).Return(nil).MinTimes(2)
	mockKV.EXPECT().Get(userIndexKey, gomock.Any()).SetArg(1, []string{msg.ID}).Return(nil).MinTimes(1)
	mockKV.EXPECT().Set(userIndexKey, gomock.Eq([]string{})).Return(true, nil).MinTimes(1)
	mockKV.EXPECT().Delete(msgKey).Return(nil).MinTimes(1)
	mockPoster.EXPECT().CreatePost(gomock.Any()).Return(nil).MinTimes(1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		s.Start() // This blocks until Stop() is called
		wg.Done()
	}()

	time.Sleep(10 * time.Millisecond)
	s.Stop()
	wg.Wait()
}

func TestProcessDueMessages_LoadMessageError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPoster := mock.NewMockPostService(ctrl)
	mockKV := mock.NewMockKVService(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)

	st := store.NewKVStore(testutil.FakeLogger{}, mockKV, mm.ListMatchingService{}, constants.MaxUserMessages)
	clk := testutil.FakeClock{NowTime: time.Now().UTC()}
	s := New(testutil.FakeLogger{}, mockPoster, st, mockChannel, "bot", clk)

	now := clk.Now()
	msgID := "uuid-5"
	postAt := now.Add(-time.Minute)
	msgForList := &types.ScheduledMessage{ID: msgID, PostAt: postAt}
	msgKey := testutil.SchedKey(msgID)

	mockKV.EXPECT().ListKeys(0, constants.MaxUserMessages, gomock.Any()).Return([]string{msgKey}, nil)
	mockKV.EXPECT().Get(msgKey, gomock.Any()).SetArg(1, *msgForList).Return(nil)
	mockKV.EXPECT().Get(msgKey, gomock.Any()).Return(errors.New("cannot load"))

	s.processDueMessages()
}

func TestProcessDueMessages_DeleteScheduleError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPoster := mock.NewMockPostService(ctrl)
	mockKV := mock.NewMockKVService(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)

	st := store.NewKVStore(testutil.FakeLogger{}, mockKV, mm.ListMatchingService{}, constants.MaxUserMessages)
	clk := testutil.FakeClock{NowTime: time.Now().UTC()}
	s := New(testutil.FakeLogger{}, mockPoster, st, mockChannel, "bot", clk)

	now := clk.Now()
	msg := &types.ScheduledMessage{
		ID: "uuid-6", UserID: "u", ChannelID: "c",
		PostAt: now.Add(-time.Minute), MessageContent: "x", Timezone: "UTC",
	}
	msgKey := testutil.SchedKey(msg.ID)

	mockKV.EXPECT().ListKeys(0, constants.MaxUserMessages, gomock.Any()).Return([]string{msgKey}, nil)
	mockKV.EXPECT().Get(msgKey, gomock.Any()).SetArg(1, *msg).Return(nil).Times(2)
	mockKV.EXPECT().Delete(msgKey).Return(errors.New("kv fail"))

	s.processDueMessages()
}

func TestProcessDueMessages_DMError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPoster := mock.NewMockPostService(ctrl)
	mockKV := mock.NewMockKVService(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)

	st := store.NewKVStore(testutil.FakeLogger{}, mockKV, mm.ListMatchingService{}, constants.MaxUserMessages)
	clk := testutil.FakeClock{NowTime: time.Now().UTC()}
	s := New(testutil.FakeLogger{}, mockPoster, st, mockChannel, "bot", clk)

	now := clk.Now()
	msg := &types.ScheduledMessage{
		ID: "uuid-7", UserID: "u", ChannelID: "c",
		PostAt: now.Add(-time.Minute), MessageContent: "x", Timezone: "UTC",
	}
	msgKey := testutil.SchedKey(msg.ID)
	userIndexKey := testutil.IndexKey(msg.UserID)
	postErr := errors.New("post fail")
	dmErr := errors.New("dm fail")
	channelInfo := &ports.ChannelInfo{ChannelID: msg.ChannelID, ChannelLink: "some-link"}

	mockKV.EXPECT().ListKeys(0, constants.MaxUserMessages, gomock.Any()).Return([]string{msgKey}, nil)
	mockKV.EXPECT().Get(msgKey, gomock.Any()).SetArg(1, *msg).Return(nil).Times(2)
	mockKV.EXPECT().Get(userIndexKey, gomock.Any()).SetArg(1, []string{msg.ID}).Return(nil)
	mockKV.EXPECT().Set(userIndexKey, gomock.Eq([]string{})).Return(true, nil)
	mockKV.EXPECT().Delete(msgKey).Return(nil)
	mockPoster.EXPECT().CreatePost(gomock.Any()).Return(postErr)
	mockChannel.EXPECT().GetInfoOrUnknown(msg.ChannelID).Return(channelInfo)
	mockChannel.EXPECT().MakeChannelLink(channelInfo).Return("in channel: some-link")
	mockPoster.EXPECT().DM("bot", msg.UserID, gomock.Any()).Return(dmErr)

	s.processDueMessages()
}

func TestProcessDueMessages_EmptyIDMap(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockPoster := mock.NewMockPostService(ctrl)
	mockKV := mock.NewMockKVService(ctrl)
	mockChannel := mock.NewMockChannelService(ctrl)

	st := store.NewKVStore(testutil.FakeLogger{}, mockKV, mm.ListMatchingService{}, constants.MaxUserMessages)
	clk := testutil.FakeClock{NowTime: time.Now().UTC()}
	s := New(testutil.FakeLogger{}, mockPoster, st, mockChannel, "bot", clk)

	mockKV.EXPECT().ListKeys(0, constants.MaxUserMessages, gomock.Any()).Return([]string{}, nil)

	s.processDueMessages()
}
