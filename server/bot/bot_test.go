package bot

import (
	"fmt"
	"testing"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mock"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type fakeImgSvc struct {
	pathCalled string
	opt        pluginapi.EnsureBotOption
}

func (f *fakeImgSvc) ProfileImagePath(p string) pluginapi.EnsureBotOption {
	f.pathCalled = p
	return f.opt
}

func TestEnsureBot(t *testing.T) {
	sentinelOpt := pluginapi.ProfileImagePath("sentinel")
	t.Run("success", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		svc := mock.NewMockBotService(ctrl)
		img := &fakeImgSvc{opt: sentinelOpt}
		svc.EXPECT().
			EnsureBot(gomock.AssignableToTypeOf(&model.Bot{}), gomock.Any()).
			DoAndReturn(func(b *model.Bot, _ pluginapi.EnsureBotOption) (string, error) {
				if b.Username != "poor-mans-scheduled-messages" {
					t.Fatalf("unexpected username %s", b.Username)
				}
				if b.DisplayName != "Message Scheduler" {
					t.Fatalf("unexpected display name %s", b.DisplayName)
				}
				if b.Description != "Poor Man's Scheduled Messages Bot" {
					t.Fatalf("unexpected description %s", b.Description)
				}
				return "bot123", nil
			})
		id, err := EnsureBot(svc, img)
		if err != nil {
			t.Fatalf("expected nil error got %v", err)
		}
		if id != "bot123" {
			t.Fatalf("expected bot123 got %s", id)
		}
		if img.pathCalled != "assets/profile.png" {
			t.Fatalf("expected assets/profile.png got %s", img.pathCalled)
		}
	})

	t.Run("error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		svc := mock.NewMockBotService(ctrl)
		img := &fakeImgSvc{opt: sentinelOpt}
		svc.EXPECT().
			EnsureBot(gomock.Any(), gomock.Any()).
			Return("", fmt.Errorf("boom"))
		id, err := EnsureBot(svc, img)
		if err == nil {
			t.Fatalf("expected error got nil")
		}
		if id != "" {
			t.Fatalf("expected empty id got %s", id)
		}
		if img.pathCalled != "assets/profile.png" {
			t.Fatalf("expected assets/profile.png got %s", img.pathCalled)
		}
	})
}
