package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
)

type Scheduler struct {
	api   *pluginapi.Client
	store store.Store
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
}

func New(api *pluginapi.Client, store store.Store) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		api:    api,
		store:  store,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Scheduler) Start() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkAndPostDueMessages()
		}
	}
}

func (s *Scheduler) Stop() {
	s.cancel()
}

func (s *Scheduler) checkAndPostDueMessages() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC().Unix()
	ids, err := s.store.ListAllScheduledIDs(s.ctx)
	if err != nil {
		s.api.Log.Error("failed to list scheduled IDs", "err", err.Error())
		return
	}

	for id, ts := range ids {
		if ts <= now {
			msg, err := s.store.GetScheduledMessage(s.ctx, id)
			if err != nil {
				s.api.Log.Warn("unable to load scheduled message", "id", id, "err", err.Error())
				continue
			}

			post := &model.Post{
				ChannelId: msg.ChannelID,
				Message:   msg.MessageContent,
				UserId:    msg.UserID,
			}

			_, err = s.api.Post.CreatePost(post)
			if err != nil {
				s.api.Log.Error("failed to post scheduled message", "id", id, "err", err.Error())
			} else {
				s.api.Log.Info("posted scheduled message", "id", id)
			}

			_ = s.store.DeleteScheduledMessage(s.ctx, id)
			_ = s.store.RemoveUserMessageID(s.ctx, msg.UserID, msg.ID)
		}
	}
}

