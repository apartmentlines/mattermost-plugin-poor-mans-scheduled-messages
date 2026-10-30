package command

import (
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/formatter"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestListReloadsRecipientPreferencesForExistingJobs(t *testing.T) {
	ctrl := gomock.NewController(t)
	st := mock.NewMockStore(ctrl)
	ch := mock.NewMockChannelService(ctrl)
	settings := types.DisplaySettings{Locale: "ru", Use24Hour: true}
	prefs := testutil.DisplayPreferences(func(userID string) types.DisplaySettings {
		require.Equal(t, "owner", userID)
		return settings
	})
	svc := NewListService(testutil.FakeLogger{}, st, ch, prefs)
	msg := &types.ScheduledMessage{ID: "job", UserID: "owner", ChannelID: "dm", RootID: "thread", Timezone: "Europe/Kaliningrad", PostAt: time.Date(2026, 10, 29, 23, 10, 0, 0, time.UTC), MessageContent: "unchanged message"}
	info := &ports.ChannelInfo{ChannelID: "dm", ChannelType: model.ChannelTypeDirect, ChannelLink: "@person"}
	st.EXPECT().ListUserMessageIDs("owner").Return([]string{"job"}, nil).Times(2)
	st.EXPECT().GetScheduledMessage("job").Return(msg, nil).Times(2)
	ch.EXPECT().GetInfoOrUnknown("dm").Return(info).Times(2)
	ch.EXPECT().MakeChannelLink(info, gomock.Any()).DoAndReturn(func(info *ports.ChannelInfo, display types.DisplaySettings) string {
		return (formatter.Renderer{DisplaySettings: display}).Destination(info)
	}).Times(2)
	response := svc.Build("owner")
	require.Equal(t, "### Запланированные сообщения", response.Text)
	atts := response.Props["attachments"].([]*model.MessageAttachment)
	require.Equal(t, "##### 30 октября 2026 01:10\nв личной переписке с: @person (в ветке)\n\nunchanged message", atts[0].Text)
	require.Equal(t, "Отправить", atts[0].Actions[0].Name)
	require.Equal(t, "Удалить", atts[0].Actions[1].Name)
	settings = types.DisplaySettings{Locale: "en", Use24Hour: false}
	response = svc.Build("owner")
	atts = response.Props["attachments"].([]*model.MessageAttachment)
	require.Contains(t, atts[0].Text, "Oct 30, 2026 1:10 AM")
	require.Contains(t, atts[0].Text, "in direct message with: @person (thread)")
	require.Equal(t, "Send", atts[0].Actions[0].Name)
	require.Equal(t, "Delete", atts[0].Actions[1].Name)
}

func TestScheduleConfirmationUsesRecipientPreferences(t *testing.T) {
	svc, mocks := setupScheduleServiceTest(t)
	svc.presentation = testutil.DisplayPreferences(func(userID string) types.DisplaySettings {
		require.Equal(t, testUserID, userID)
		return types.DisplaySettings{Locale: "ru", Use24Hour: true}
	})
	info := &ports.ChannelInfo{ChannelID: testChannelID, ChannelType: model.ChannelTypeOpen, ChannelLink: "~town-square"}
	mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return(nil, nil)
	mocks.userAPI.EXPECT().Get(testUserID).Return(&model.User{Timezone: map[string]string{"manualTimezone": "UTC"}}, nil)
	mocks.store.EXPECT().GenerateMessageID().Return("job")
	mocks.store.EXPECT().SaveScheduledMessage(testUserID, gomock.Any()).DoAndReturn(func(_ string, msg *types.ScheduledMessage) error {
		require.Equal(t, "body", msg.MessageContent)
		require.Equal(t, time.Date(2026, 10, 30, 13, 10, 0, 0, time.UTC), msg.PostAt)
		return nil
	})
	mocks.channel.EXPECT().GetInfoOrUnknown(testChannelID).Return(info)
	mocks.channel.EXPECT().MakeChannelLink(info, types.DisplaySettings{Locale: "ru", Use24Hour: true}).Return("в канале: ~town-square")
	response := svc.Build(defaultArgs(), "at 1:10PM on 2026-10-30 message body")
	require.Equal(t, "✅ Сообщение запланировано на 30 октября 2026 13:10 (UTC) в канале: ~town-square", response.Text)
}

func TestListInvalidTimezoneFallsBackToUTC(t *testing.T) {
	ctrl := gomock.NewController(t)
	ch := mock.NewMockChannelService(ctrl)
	svc := &ListService{logger: testutil.FakeLogger{}, channel: ch}
	ch.EXPECT().GetInfoOrUnknown("channel").Return(&ports.ChannelInfo{})
	ch.EXPECT().MakeChannelLink(gomock.Any(), gomock.Any()).Return("N/A")
	atts := svc.buildAttachments([]*types.ScheduledMessage{{ID: "job", ChannelID: "channel", Timezone: "invalid", PostAt: time.Date(2026, 10, 30, 13, 10, 0, 0, time.UTC)}}, formatter.Renderer{DisplaySettings: types.DisplaySettings{Locale: "ru", Use24Hour: true}})
	require.Contains(t, atts[0].Text, "30 октября 2026 13:10")
}

func TestPastScheduleErrorUsesPersonalDateAndClock(t *testing.T) {
	for _, tc := range []struct {
		military bool
		want     string
	}{
		{true, "15 января 2024 09:10"},
		{false, "15 января 2024 9:10 AM"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			svc, mocks := setupScheduleServiceTest(t)
			svc.presentation = testutil.DisplayPreferences(func(string) types.DisplaySettings {
				return types.DisplaySettings{Locale: "ru", Use24Hour: tc.military}
			})
			mocks.store.EXPECT().ListUserMessageIDs(testUserID).Return(nil, nil)
			mocks.userAPI.EXPECT().Get(testUserID).Return(&model.User{Timezone: map[string]string{"manualTimezone": "UTC"}}, nil)
			response := svc.Build(defaultArgs(), "at 9:10AM on 2024-01-15 message body")
			require.Contains(t, response.Text, "Время отправки "+tc.want+" уже прошло.")
		})
	}
}
