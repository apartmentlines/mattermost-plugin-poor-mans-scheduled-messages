package command

import (
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestDeliveryStateControlsVisibleActions(t *testing.T) {
	for _, tc := range []struct {
		state   types.DeliveryState
		actions []string
	}{
		{types.StateCreating, []string{"delete"}},
		{types.StatePending, []string{"send", "delete"}},
		{types.StateProcessing, []string{"delete"}},
		{types.StateSending, nil}, {types.StateReconciling, nil},
		{types.StateUncertain, []string{"send"}},
		{types.StateRetry, []string{"send", "delete"}},
		{types.StateFailed, []string{"send", "delete"}},
		{types.StateSent, nil}, {types.StateCancelled, nil},
	} {
		t.Run(string(tc.state), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			st := mock.NewMockStore(ctrl)
			ch := mock.NewMockChannelService(ctrl)
			msg := &types.ScheduledMessage{ID: "job", UserID: "user", ChannelID: "channel", State: tc.state, Attempts: 2, LastReason: "rate_limited", MessageContent: "text", Timezone: "invalid/timezone", PostAt: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)}
			st.EXPECT().ListUserMessageIDs("user").Return([]string{msg.ID}, nil)
			st.EXPECT().GetScheduledMessage(msg.ID).Return(msg, nil)
			if msg.Terminal() {
				st.EXPECT().CleanupMessageFromUserIndex(msg.UserID, msg.ID).Return(nil)
			} else {
				info := &ports.ChannelInfo{ChannelID: msg.ChannelID}
				ch.EXPECT().GetInfoOrUnknown(msg.ChannelID).Return(info)
				ch.EXPECT().MakeChannelLink(info, gomock.Any()).Return("channel")
			}
			resp := NewListService(testutil.FakeLogger{}, st, ch, nil).Build(msg.UserID)
			if msg.Terminal() {
				require.Equal(t, constants.EmptyListMessage, resp.Text)
				return
			}
			atts := resp.Props["attachments"].([]*model.MessageAttachment)
			require.Len(t, atts, 1)
			require.Contains(t, atts[0].Text, string(tc.state))
			require.Contains(t, atts[0].Text, "attempts 2/5")
			var ids []string
			for _, a := range atts[0].Actions {
				ids = append(ids, a.Id)
			}
			require.Equal(t, tc.actions, ids)
			if tc.state == types.StateUncertain {
				require.Equal(t, "Check delivery", atts[0].Actions[0].Name)
			}
		})
	}
}

func TestDeliveryCheckUsesLocaleAndPersonalClock(t *testing.T) {
	for _, tc := range []struct {
		locale     string
		military   bool
		wantDate   string
		wantNext   string
		wantAction string
	}{
		{"ru", true, "30 октября 2026 01:10", "Следующая проверка: ", "Проверить доставку"},
		{"ru", false, "30 октября 2026 1:10 AM", "Следующая проверка: ", "Проверить доставку"},
		{"en", true, "Oct 30, 2026 01:10", "Next check: ", "Check delivery"},
		{"en", false, "Oct 30, 2026 1:10 AM", "Next check: ", "Check delivery"},
	} {
		t.Run(tc.wantDate, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			st := mock.NewMockStore(ctrl)
			ch := mock.NewMockChannelService(ctrl)
			display := types.DisplaySettings{Locale: tc.locale, Use24Hour: tc.military}
			prefs := testutil.DisplayPreferences(func(userID string) types.DisplaySettings {
				require.Equal(t, "user", userID)
				return display
			})
			msg := &types.ScheduledMessage{
				ID: "job", UserID: "user", ChannelID: "channel", State: types.StateUncertain,
				MessageContent: "text", Timezone: "Europe/Kaliningrad", Attempts: 2,
				PostAt:        time.Date(2026, 10, 29, 23, 0, 0, 0, time.UTC),
				NextAttemptAt: time.Date(2026, 10, 29, 23, 10, 0, 0, time.UTC),
			}
			st.EXPECT().ListUserMessageIDs("user").Return([]string{msg.ID}, nil)
			st.EXPECT().GetScheduledMessage(msg.ID).Return(msg, nil)
			info := &ports.ChannelInfo{ChannelID: msg.ChannelID}
			ch.EXPECT().GetInfoOrUnknown(msg.ChannelID).Return(info)
			ch.EXPECT().MakeChannelLink(info, display).Return("channel")
			resp := NewListService(testutil.FakeLogger{}, st, ch, prefs).Build("user")
			atts := resp.Props["attachments"].([]*model.MessageAttachment)
			require.Len(t, atts, 1)
			require.Contains(t, atts[0].Text, tc.wantNext+tc.wantDate)
			require.Len(t, atts[0].Actions, 1)
			require.Equal(t, "send", atts[0].Actions[0].Id)
			require.Equal(t, tc.wantAction, atts[0].Actions[0].Name)
		})
	}
}
