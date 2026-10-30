package scheduler

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func setupDelivery(t *testing.T) (*Scheduler, ports.Store, *testutil.MemoryKV, *mock.MockPostService, *types.ScheduledMessage, *testutil.FakeClock) {
	t.Helper()
	ctrl := gomock.NewController(t)
	poster := mock.NewMockPostService(ctrl)
	ch := mock.NewMockChannelService(ctrl)
	info := &ports.ChannelInfo{ChannelID: "channel"}
	ch.EXPECT().GetInfoOrUnknown("channel").Return(info).AnyTimes()
	ch.EXPECT().MakeChannelLink(info, gomock.Any()).Return("channel").AnyTimes()
	poster.EXPECT().DM("bot", "user", gomock.Any()).Return(nil).AnyTimes()
	db := testutil.NewMemoryKV()
	st := store.NewKVStore(testutil.FakeLogger{}, &pluginapi.NewClient(db, nil).KV, nil, 100)
	clk := &testutil.FakeClock{NowTime: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)}
	msg := &types.ScheduledMessage{ID: "job", UserID: "user", ChannelID: "channel", RootID: "thread", MessageContent: "text", PostAt: clk.Now(), Timezone: "UTC"}
	require.NoError(t, st.SaveScheduledMessage(msg.UserID, msg))
	return New(testutil.FakeLogger{}, poster, st, ch, "bot", clk, nil), st, db, poster, msg, clk
}

func TestSendNowSuccessKeepsReceiptAndRejectsStaleSend(t *testing.T) {
	s, st, _, poster, msg, _ := setupDelivery(t)
	poster.EXPECT().CreatePost(gomock.Any()).DoAndReturn(func(post *model.Post) error {
		require.Equal(t, msg.MessageContent, post.Message)
		require.Equal(t, msg.RootID, post.RootId)
		require.Equal(t, msg.ID, post.GetProp(types.ScheduledJobProp))
		require.NotEmpty(t, post.PendingPostId)
		post.Id = "post"
		return nil
	})
	require.NoError(t, s.SendNow(msg))
	got, err := st.GetScheduledMessage(msg.ID)
	require.NoError(t, err)
	require.Equal(t, types.StateSent, got.State)
	require.Equal(t, "post", got.PostID)
	require.Empty(t, got.MessageContent)
	require.ErrorIs(t, s.SendNow(msg), types.ErrMessageFinal)
	ids, err := st.ListUserMessageIDs(msg.UserID)
	require.NoError(t, err)
	require.Empty(t, ids)
}

func TestManualAndTimerShareAtomicClaim(t *testing.T) {
	for _, timerFirst := range []bool{false, true} {
		t.Run(map[bool]string{true: "timer", false: "manual"}[timerFirst], func(t *testing.T) {
			s, st, db, poster, msg, clk := setupDelivery(t)
			started := make(chan struct{})
			release := make(chan struct{})
			poster.EXPECT().CreatePost(gomock.Any()).DoAndReturn(func(post *model.Post) error { close(started); <-release; post.Id = "post"; return nil })
			finished := make(chan error, 1)
			go func() {
				if timerFirst {
					s.processDueMessages()
					finished <- nil
				} else {
					finished <- s.SendNow(msg)
				}
			}()
			<-started
			otherStore := store.NewKVStore(testutil.FakeLogger{}, &pluginapi.NewClient(db, nil).KV, nil, 100)
			other := New(testutil.FakeLogger{}, poster, otherStore, s.linker, "bot", clk, nil)
			if timerFirst {
				require.ErrorIs(t, other.SendNow(msg), types.ErrMessageBusy)
			} else {
				other.processDueMessages()
			}
			require.ErrorIs(t, st.DeleteScheduledMessage(msg.UserID, msg.ID), types.ErrMessageBusy)
			close(release)
			require.NoError(t, <-finished)
		})
	}
}

func TestRestartAfterPublicationBeforeReceiptDoesNotRepublish(t *testing.T) {
	s, st, db, poster, msg, clk := setupDelivery(t)
	var published *model.Post
	poster.EXPECT().CreatePost(gomock.Any()).DoAndReturn(func(post *model.Post) error {
		post.Id = "post"
		published = post
		db.FailWrites(constants.SchedPrefix+msg.ID, 1)
		return nil
	})
	require.Error(t, s.SendNow(msg))
	got, err := st.GetScheduledMessage(msg.ID)
	require.NoError(t, err)
	require.Equal(t, types.StateSending, got.State)
	clk.NowTime = clk.Now().Add(types.DeliveryLease)
	poster.EXPECT().GetPostsForChannel(msg.ChannelID, 0, 100).Return(&model.PostList{Order: []string{published.Id}, Posts: map[string]*model.Post{published.Id: published}}, nil)
	restarted := New(testutil.FakeLogger{}, poster, store.NewKVStore(testutil.FakeLogger{}, &pluginapi.NewClient(db, nil).KV, nil, 100), s.linker, "bot", clk, nil)
	restarted.processDueMessages()
	got, err = st.GetScheduledMessage(msg.ID)
	require.NoError(t, err)
	require.Equal(t, types.StateSent, got.State)
	require.Equal(t, published.Id, got.PostID)
}

func TestTimeoutAfterCommitReconcilesWithoutRepublishing(t *testing.T) {
	s, st, _, poster, msg, _ := setupDelivery(t)
	var published *model.Post
	poster.EXPECT().CreatePost(gomock.Any()).DoAndReturn(func(post *model.Post) error { post.Id = "post"; published = post; return errors.New("lost response") })
	require.Error(t, s.SendNow(msg))
	got, err := st.GetScheduledMessage(msg.ID)
	require.NoError(t, err)
	require.Equal(t, types.StateUncertain, got.State)
	poster.EXPECT().GetPostsForChannel(msg.ChannelID, 0, 100).Return(&model.PostList{Order: []string{"post"}, Posts: map[string]*model.Post{"post": published}}, nil)
	require.NoError(t, s.SendNow(msg))
	got, err = st.GetScheduledMessage(msg.ID)
	require.NoError(t, err)
	require.Equal(t, types.StateSent, got.State)
}

func TestUnknownOutcomeWithoutPostRemainsDurable(t *testing.T) {
	s, st, _, poster, msg, _ := setupDelivery(t)
	poster.EXPECT().CreatePost(gomock.Any()).Return(&model.AppError{StatusCode: 503})
	require.Error(t, s.SendNow(msg))
	poster.EXPECT().GetPostsForChannel(msg.ChannelID, 0, 100).Return(&model.PostList{}, nil).Times(2)
	for i := 0; i < 2; i++ {
		require.ErrorIs(t, s.SendNow(msg), types.ErrDeliveryUncertain)
	}
	got, err := st.GetScheduledMessage(msg.ID)
	require.NoError(t, err)
	require.Equal(t, types.StateUncertain, got.State)
	require.Equal(t, msg.MessageContent, got.MessageContent)
	require.Equal(t, 1, got.Attempts)
}

func TestRateLimitedDeliveryRetriesWithBackoffAndLimit(t *testing.T) {
	s, st, _, poster, msg, clk := setupDelivery(t)
	poster.EXPECT().CreatePost(gomock.Any()).Return(&model.AppError{StatusCode: 429}).Times(types.MaxDeliveryAttempts)
	for attempt := 1; attempt <= types.MaxDeliveryAttempts; attempt++ {
		s.processDueMessages()
		got, err := st.GetScheduledMessage(msg.ID)
		require.NoError(t, err)
		require.Equal(t, attempt, got.Attempts)
		if attempt < types.MaxDeliveryAttempts {
			require.Equal(t, types.StateRetry, got.State)
			s.processDueMessages() // Too early: no extra call to CreatePost.
			clk.NowTime = got.NextAttemptAt
		} else {
			require.Equal(t, types.StateFailed, got.State)
		}
	}
	s.processDueMessages()
}

func TestExpiredProcessingClaimCanBeSafelyRetried(t *testing.T) {
	s, st, _, poster, msg, clk := setupDelivery(t)
	_, err := st.ClaimScheduledMessage(msg.UserID, msg.ID, clk.Now(), true)
	require.NoError(t, err)
	clk.NowTime = clk.Now().Add(types.DeliveryLease)
	poster.EXPECT().CreatePost(gomock.Any()).DoAndReturn(func(post *model.Post) error { post.Id = "post"; return nil })
	s.processDueMessages()
	got, err := st.GetScheduledMessage(msg.ID)
	require.NoError(t, err)
	require.Equal(t, types.StateSent, got.State)
}

func TestStartFailureNeverPublishes(t *testing.T) {
	s, st, db, _, msg, clk := setupDelivery(t)
	claim, err := st.ClaimScheduledMessage(msg.UserID, msg.ID, clk.Now(), true)
	require.NoError(t, err)
	db.FailWrites(constants.SchedPrefix+msg.ID, 1)
	require.Error(t, st.StartDelivery(claim, clk.Now()))
	clk.NowTime = clk.Now().Add(time.Minute)
	s.processDueMessages() // Lease is still active, no publication.
}

func TestSchedulerRunCanStop(t *testing.T) {
	s, _, _, _, _, _ := setupDelivery(t)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); s.run() }()
	s.Stop()
	wg.Wait()
}
