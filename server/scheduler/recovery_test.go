package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mm"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	testifymock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSendNowPublicationFailureRetainsJob(t *testing.T) {
	ctrl := gomock.NewController(t)
	poster := mock.NewMockPostService(ctrl)
	ch := mock.NewMockChannelService(ctrl)
	db := testutil.NewMemoryKV()
	cli := pluginapi.NewClient(db, nil)
	st := store.NewKVStore(testutil.FakeLogger{}, &cli.KV, mm.NewListMatchingService(), 100)
	msg := &types.ScheduledMessage{ID: "job", UserID: "author", ChannelID: "channel", MessageContent: "text", Timezone: "UTC", PostAt: time.Now()}
	require.NoError(t, st.SaveScheduledMessage(msg.UserID, msg))
	poster.EXPECT().CreatePost(gomock.Any()).Return(&model.AppError{Id: "test.invalid", StatusCode: 400})
	info := &ports.ChannelInfo{ChannelID: msg.ChannelID}
	ch.EXPECT().GetInfoOrUnknown(msg.ChannelID).Return(info).AnyTimes()
	ch.EXPECT().MakeChannelLink(info, gomock.Any()).Return("channel").AnyTimes()
	poster.EXPECT().DM("bot", msg.UserID, gomock.Any()).Return(nil).AnyTimes()
	s := New(testutil.FakeLogger{}, poster, st, ch, "bot", testutil.FakeClock{NowTime: time.Now()}, nil)
	require.Error(t, s.SendNow(msg))
	saved, err := st.GetScheduledMessage(msg.ID)
	require.NoError(t, err, "a publication error must not delete the job")
	require.Equal(t, msg.MessageContent, saved.MessageContent)
}

func TestNormalizedNotFoundAllowsCancellationOrExplicitRetry(t *testing.T) {
	for _, action := range []string{"cancel", "retry"} {
		t.Run(action, func(t *testing.T) {
			s, st, _, poster, msg, _ := setupDelivery(t)
			api := &plugintest.API{}
			api.On("CreatePost", testifymock.Anything).Return(nil, &model.AppError{
				Id: "test.missing_channel", StatusCode: 404,
			}).Once()
			client := pluginapi.NewClient(api, nil)
			// Exercise the real adapter's 404 -> ErrNotFound normalization.
			poster.EXPECT().CreatePost(gomock.Any()).DoAndReturn(client.Post.CreatePost)
			require.ErrorIs(t, s.SendNow(msg), pluginapi.ErrNotFound)
			saved, err := st.GetScheduledMessage(msg.ID)
			require.NoError(t, err)
			require.Equal(t, types.StateFailed, saved.State)
			require.Equal(t, "publication_rejected_404", saved.LastReason)
			require.Equal(t, msg.MessageContent, saved.MessageContent)
			if action == "cancel" {
				require.NoError(t, st.DeleteScheduledMessage(msg.UserID, msg.ID))
			} else {
				poster.EXPECT().CreatePost(gomock.Any()).DoAndReturn(func(post *model.Post) error {
					post.Id = "confirmed-post"
					return nil
				})
				require.NoError(t, s.SendNow(msg))
			}
			api.AssertExpectations(t)
		})
	}
}

func TestWrappedNormalizedNotFoundIsAConfirmedRejection(t *testing.T) {
	err := fmt.Errorf("publication failed: %w", pluginapi.ErrNotFound)
	result := classifyFailure(err)
	require.Equal(t, types.StateFailed, result.State)
	require.Equal(t, "publication_rejected_404", result.Reason)
}
