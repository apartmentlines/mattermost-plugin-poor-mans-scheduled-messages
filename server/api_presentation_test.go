package main

import (
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestActionConfirmationsUseCurrentRecipientPreferences(t *testing.T) {
	ctrl := gomock.NewController(t)
	p, posts, channels, _, _ := setupPluginForAPI(t, ctrl)
	settings := types.DisplaySettings{Locale: "ru", Use24Hour: true}
	p.presentation = testutil.DisplayPreferences(func(userID string) types.DisplaySettings {
		require.Equal(t, "owner", userID)
		return settings
	})
	msg := &types.ScheduledMessage{ID: "job", ChannelID: "channel", Timezone: "Europe/Kaliningrad", PostAt: time.Date(2026, 10, 29, 23, 10, 0, 0, time.UTC)}
	channels.EXPECT().GetInfoOrUnknown("channel").Return(nil).Times(2)
	channels.EXPECT().MakeChannelLink(nil, settings).Return("в канале: ~town-square")
	posts.EXPECT().SendEphemeralPost("owner", gomock.Any()).Do(func(_ string, post *model.Post) {
		require.Equal(t, "✅ Сообщение, запланированное на **30 октября 2026 01:10** в канале: ~town-square, отправлено.", post.Message)
	})
	p.sendSendConfirmation("owner", "channel", msg)
	settings = types.DisplaySettings{Locale: "en", Use24Hour: false}
	channels.EXPECT().MakeChannelLink(nil, settings).Return("in channel: ~town-square")
	posts.EXPECT().SendEphemeralPost("owner", gomock.Any()).Do(func(_ string, post *model.Post) {
		require.Equal(t, "✅ Message scheduled for **Oct 30, 2026 1:10 AM** in channel: ~town-square has been deleted.", post.Message)
	})
	p.sendDeletionConfirmation("owner", "channel", msg)
	settings = types.DisplaySettings{Locale: "ru"}
	updated := p.buildEphemeralListUpdate("owner", "post", "channel", &model.CommandResponse{})
	require.Equal(t, "У вас нет запланированных сообщений.", updated.Message)
}
