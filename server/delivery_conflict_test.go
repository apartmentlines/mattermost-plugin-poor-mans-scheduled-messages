package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/command"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/scheduler"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHTTPActionsConflictWithPublicationInProgress(t *testing.T) {
	for _, action := range []string{"send", "delete"} {
		t.Run(action, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			poster := mock.NewMockPostService(ctrl)
			ch := mock.NewMockChannelService(ctrl)
			info := &ports.ChannelInfo{ChannelID: "channel"}
			ch.EXPECT().GetInfoOrUnknown("channel").Return(info)
			ch.EXPECT().MakeChannelLink(info, gomock.Any()).Return("channel")
			db := testutil.NewMemoryKV()
			st := store.NewKVStore(testutil.FakeLogger{}, &pluginapi.NewClient(db, nil).KV, nil, 100)
			clk := testutil.FakeClock{NowTime: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)}
			msg := &types.ScheduledMessage{ID: "job", UserID: "user", ChannelID: "channel", MessageContent: "text", Timezone: "UTC", PostAt: clk.Now()}
			require.NoError(t, st.SaveScheduledMessage(msg.UserID, msg))
			claim, err := st.ClaimScheduledMessage(msg.UserID, msg.ID, clk.Now(), true)
			require.NoError(t, err)
			require.NoError(t, st.StartDelivery(claim, clk.Now()))
			list := command.NewListService(testutil.FakeLogger{}, st, ch, nil)
			handler := command.NewHandler(testutil.FakeLogger{}, nil, nil, st, ch, list, nil, "")
			s := scheduler.New(testutil.FakeLogger{}, poster, st, ch, "bot", clk, nil)
			p := &Plugin{logger: testutil.FakeLogger{}, poster: poster, Command: handler, Scheduler: s}
			poster.EXPECT().UpdateEphemeralPost(msg.UserID, gomock.Any())
			poster.EXPECT().SendEphemeralPost(msg.UserID, gomock.Any()).Do(func(_ string, post *model.Post) { require.Contains(t, post.Message, "already being processed") })
			req := createSendRequest(t, msg.UserID, "list-post", "channel", action, msg.ID)
			if action == "delete" {
				req = createDeleteRequest(t, msg.UserID, "list-post", "channel", action, msg.ID)
			}
			rr := httptest.NewRecorder()
			p.ServeHTTP(nil, rr, req)
			require.Equal(t, http.StatusConflict, rr.Code)
			saved, err := st.GetScheduledMessage(msg.ID)
			require.NoError(t, err)
			require.Equal(t, types.StateSending, saved.State)
			require.Equal(t, claim.ClaimToken, saved.ClaimToken)
			require.Equal(t, msg.MessageContent, saved.MessageContent)
		})
	}
}
