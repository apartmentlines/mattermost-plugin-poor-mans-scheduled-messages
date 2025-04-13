package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type Scheduler struct {
	api    *pluginapi.Client
	store  store.Store
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
	ids, listErr := s.store.ListAllScheduledIDs()
	if listErr != nil {
		s.api.Log.Error("failed to list scheduled IDs", "err", listErr.Error())
		return
	}

	for id, ts := range ids {
		if ts <= now {
			msg, getErr := s.store.GetScheduledMessage(id)
			if getErr != nil {
				s.api.Log.Warn("unable to load scheduled message", "id", id, "err", getErr.Error())
				continue
			}

			post := &model.Post{
				ChannelId: msg.ChannelID,
				Message:   msg.MessageContent,
				UserId:    msg.UserID,
			}

			deleteErr := s.store.DeleteScheduledMessage(msg.UserID, msg.ID)
			if deleteErr != nil {
				s.api.Log.Error("failed to delete scheduled message from store", "id", id, "err", deleteErr.Error())
				// TODO: send error to user
				continue
			}

			postErr := s.api.Post.CreatePost(post)
			if postErr != nil {
				s.api.Log.Error("failed to post scheduled message", "id", id, "err", postErr.Error())
				// TODO: send error to user
			} else {
				s.api.Log.Info("posted scheduled message", "id", id)
			}
		}
	}
}
