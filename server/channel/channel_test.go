package channel

import (
	"errors"
	"reflect"
	"testing"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost/server/public/model"
)

func newTestChannel(t *testing.T) (*Channel,
	*mock.MockChannelDataService,
	*mock.MockTeamService,
	*mock.MockUserService,
	*gomock.Controller,
) {
	ctrl := gomock.NewController(t)
	chData := mock.NewMockChannelDataService(ctrl)
	teamSvc := mock.NewMockTeamService(ctrl)
	userSvc := mock.NewMockUserService(ctrl)

	ch := New(testutil.FakeLogger{}, chData, teamSvc, userSvc)
	return ch, chData, teamSvc, userSvc, ctrl
}

func TestGetInfo(t *testing.T) {
	t.Run("channel get error", func(t *testing.T) {
		ch, chData, _, _, ctrl := newTestChannel(t)
		defer ctrl.Finish()

		chID := "bad"
		chData.EXPECT().Get(chID).Return(nil, errors.New("boom")).Times(1)

		info, err := ch.GetInfo(chID)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if info != nil {
			t.Fatalf("expected nil ChannelInfo, got %#v", info)
		}
	})

	t.Run("direct / group happy path", func(t *testing.T) {
		ch, chData, _, userSvc, ctrl := newTestChannel(t)
		defer ctrl.Finish()

		channelID := "chan1"
		member1 := &model.ChannelMember{UserId: "uid1"}
		member2 := &model.ChannelMember{UserId: "uid2"}

		chData.EXPECT().
			Get(channelID).
			Return(&model.Channel{Id: channelID, Type: model.ChannelTypeDirect}, nil).
			Times(1)

		chData.EXPECT().
			ListMembers(channelID, 0, 100).
			Return([]*model.ChannelMember{member1, member2}, nil).
			Times(1)

		userSvc.EXPECT().Get("uid1").Return(&model.User{Id: "uid1", Username: "alice"}, nil).Times(1)
		userSvc.EXPECT().Get("uid2").Return(&model.User{Id: "uid2", Username: "bob"}, nil).Times(1)

		got, err := ch.GetInfo(channelID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := &ports.ChannelInfo{
			ChannelID:   channelID,
			ChannelType: model.ChannelTypeDirect,
			ChannelLink: "@alice, @bob",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected ChannelInfo.\nwant: %#v\ngot:  %#v", want, got)
		}
	})

	t.Run("private channel happy path", func(t *testing.T) {
		ch, chData, teamSvc, _, ctrl := newTestChannel(t)
		defer ctrl.Finish()

		channelID := "private1"
		teamID := "teamPrivate"

		chData.EXPECT().
			Get(channelID).
			Return(&model.Channel{
				Id:     channelID,
				Type:   model.ChannelTypePrivate,
				Name:   "secret-channel",
				TeamId: teamID,
			}, nil).
			Times(1)

		teamSvc.EXPECT().Get(teamID).
			Return(&model.Team{Id: teamID, DisplayName: "Secret Team"}, nil).
			Times(1)

		got, err := ch.GetInfo(channelID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := &ports.ChannelInfo{
			ChannelID:   channelID,
			ChannelType: model.ChannelTypePrivate,
			ChannelLink: "~secret-channel",
			TeamName:    "Secret Team",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected ChannelInfo.\nwant: %#v\ngot:  %#v", want, got)
		}
	})

	t.Run("group happy path", func(t *testing.T) {
		ch, chData, _, userSvc, ctrl := newTestChannel(t)
		defer ctrl.Finish()

		channelID := "group1"
		member1 := &model.ChannelMember{UserId: "uid1"}
		member2 := &model.ChannelMember{UserId: "uid2"}
		member3 := &model.ChannelMember{UserId: "uid3"}

		chData.EXPECT().
			Get(channelID).
			Return(&model.Channel{Id: channelID, Type: model.ChannelTypeGroup}, nil).
			Times(1)

		chData.EXPECT().
			ListMembers(channelID, 0, 100).
			Return([]*model.ChannelMember{member1, member2, member3}, nil).
			Times(1)

		userSvc.EXPECT().Get("uid1").Return(&model.User{Id: "uid1", Username: "alice"}, nil).Times(1)
		userSvc.EXPECT().Get("uid2").Return(&model.User{Id: "uid2", Username: "bob"}, nil).Times(1)
		userSvc.EXPECT().Get("uid3").Return(&model.User{Id: "uid3", Username: "charlie"}, nil).Times(1)

		got, err := ch.GetInfo(channelID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := &ports.ChannelInfo{
			ChannelID:   channelID,
			ChannelType: model.ChannelTypeGroup,
			ChannelLink: "@alice, @bob, @charlie",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected ChannelInfo.\nwant: %#v\ngot:  %#v", want, got)
		}
	})

	t.Run("direct path list members error", func(t *testing.T) {
		ch, chData, _, _, ctrl := newTestChannel(t)
		defer ctrl.Finish()

		channelID := "chan_lst_err"
		chData.EXPECT().Get(channelID).
			Return(&model.Channel{Id: channelID, Type: model.ChannelTypeDirect}, nil).
			Times(1)

		chData.EXPECT().
			ListMembers(channelID, 0, 100).
			Return(nil, errors.New("list err")).
			Times(1)

		info, err := ch.GetInfo(channelID)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if info != nil {
			t.Fatalf("expected nil ChannelInfo on error")
		}
	})

	t.Run("direct path user lookup error", func(t *testing.T) {
		ch, chData, _, userSvc, ctrl := newTestChannel(t)
		defer ctrl.Finish()

		channelID := "chan_user_err"
		member1 := &model.ChannelMember{UserId: "uid1"}

		chData.EXPECT().Get(channelID).
			Return(&model.Channel{Id: channelID, Type: model.ChannelTypeDirect}, nil).
			Times(1)

		chData.EXPECT().
			ListMembers(channelID, 0, 100).
			Return([]*model.ChannelMember{member1}, nil).
			Times(1)

		userSvc.EXPECT().Get("uid1").Return(nil, errors.New("user err")).Times(1)

		info, err := ch.GetInfo(channelID)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if info != nil {
			t.Fatalf("expected nil ChannelInfo on error")
		}
	})

	t.Run("public/private happy path", func(t *testing.T) {
		ch, chData, teamSvc, _, ctrl := newTestChannel(t)
		defer ctrl.Finish()

		channelID := "public1"
		teamID := "team1"

		chData.EXPECT().Get(channelID).
			Return(&model.Channel{
				Id:     channelID,
				Type:   model.ChannelTypeOpen,
				Name:   "town-square",
				TeamId: teamID,
			}, nil).
			Times(1)

		teamSvc.EXPECT().Get(teamID).
			Return(&model.Team{Id: teamID, DisplayName: "Demo Team"}, nil).
			Times(1)

		got, err := ch.GetInfo(channelID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		want := &ports.ChannelInfo{
			ChannelID:   channelID,
			ChannelType: model.ChannelTypeOpen,
			ChannelLink: "~town-square",
			TeamName:    "Demo Team",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unexpected ChannelInfo.\nwant: %#v\ngot:  %#v", want, got)
		}
	})

	t.Run("public path team fetch error", func(t *testing.T) {
		ch, chData, teamSvc, _, ctrl := newTestChannel(t)
		defer ctrl.Finish()

		channelID := "public_err"
		teamID := "team2"

		chData.EXPECT().Get(channelID).
			Return(&model.Channel{
				Id:     channelID,
				Type:   model.ChannelTypeOpen,
				Name:   "some",
				TeamId: teamID,
			}, nil).Times(1)

		teamSvc.EXPECT().Get(teamID).Return(nil, errors.New("team err")).Times(1)

		info, err := ch.GetInfo(channelID)
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if info != nil {
			t.Fatalf("expected nil ChannelInfo on error")
		}
	})
}

func TestGetInfoOrUnknown(t *testing.T) {
	t.Run("success forwards info", func(t *testing.T) {
		ch, chData, teamSvc, _, ctrl := newTestChannel(t)
		defer ctrl.Finish()

		channelID := "public_success"
		teamID := "team3"

		chData.EXPECT().Get(channelID).
			Return(&model.Channel{
				Id:     channelID,
				Type:   model.ChannelTypeOpen,
				Name:   "square",
				TeamId: teamID,
			}, nil).Times(2) // will be called twice (GetInfo then GetInfoOrUnknown)

		teamSvc.EXPECT().Get(teamID).
			Return(&model.Team{Id: teamID, DisplayName: "Demo"}, nil).
			Times(2)

		// first call
		info1, err := ch.GetInfo(channelID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// second, through GetInfoOrUnknown
		info2 := ch.GetInfoOrUnknown(channelID)

		if !reflect.DeepEqual(info1, info2) {
			t.Fatalf("expected ChannelInfo values to match.\nwant: %#v\ngot:  %#v", info1, info2)
		}
	})

	t.Run("failure returns UnknownChannel", func(t *testing.T) {
		ch, chData, _, _, ctrl := newTestChannel(t)
		defer ctrl.Finish()

		chData.EXPECT().Get("bad").Return(nil, errors.New("boom")).Times(1)

		got := ch.GetInfoOrUnknown("bad")
		if got.ChannelID != "" || got.ChannelLink != "N/A" {
			t.Fatalf("expected UnknownChannel, got %#v", got)
		}
	})
}

func TestUnknownChannel(t *testing.T) {
	ch, _, _, _, ctrl := newTestChannel(t)
	defer ctrl.Finish()

	uc := ch.UnknownChannel()
	if uc.ChannelLink != "N/A" || uc.ChannelID != "" {
		t.Fatalf("unexpected unknown channel value: %#v", uc)
	}
}

func TestMakeChannelLink(t *testing.T) {
	ch, _, _, _, ctrl := newTestChannel(t)
	defer ctrl.Finish()

	tests := []struct {
		name string
		info *ports.ChannelInfo
		want string
	}{
		{
			name: "unknown",
			info: &ports.ChannelInfo{ChannelLink: "N/A"},
			want: "N/A",
		},
		{
			name: "direct",
			info: &ports.ChannelInfo{
				ChannelID:   "d1",
				ChannelType: model.ChannelTypeDirect,
				ChannelLink: "@alice",
			},
			want: "in direct message with: @alice",
		},
		{
			name: "group",
			info: &ports.ChannelInfo{
				ChannelID:   "g1",
				ChannelType: model.ChannelTypeGroup,
				ChannelLink: "@alice, @bob",
			},
			want: "in direct message with: @alice, @bob",
		},
		{
			name: "public",
			info: &ports.ChannelInfo{
				ChannelID:   "c1",
				ChannelType: model.ChannelTypeOpen,
				ChannelLink: "~town-square",
			},
			want: "in channel: ~town-square",
		},
		{
			name: "private",
			info: &ports.ChannelInfo{
				ChannelID:   "c2",
				ChannelType: model.ChannelTypePrivate,
				ChannelLink: "~secret-channel",
			},
			want: "in channel: ~secret-channel",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := ch.MakeChannelLink(tc.info)
			if got != tc.want {
				t.Fatalf("want %q, got %q", tc.want, got)
			}
		})
	}
}
