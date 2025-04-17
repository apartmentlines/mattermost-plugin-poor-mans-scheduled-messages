package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/channel"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/clock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/formatter"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type Scheduler struct {
	api     *pluginapi.Client
	store   store.Store
	channel *channel.Channel
	botID   string
	clock   clock.Clock
	ctx     context.Context
	cancel  context.CancelFunc
	mu      sync.Mutex
}

func New(api *pluginapi.Client, store store.Store, channel *channel.Channel, botID string, clk clock.Clock) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		api:     api,
		store:   store,
		channel: channel,
		botID:   botID,
		ctx:     ctx,
		cancel:  cancel,
		clock:   clk,
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

	now := s.clock.Now().UTC().Unix()
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
				continue
			}

			postErr := s.api.Post.CreatePost(post)
			if postErr != nil {
				s.api.Log.Error("failed to post scheduled message", "id", id, "err", postErr.Error())
				s.dmUserOnFailedMessage(msg, postErr)
			} else {
				s.api.Log.Info("posted scheduled message", "id", id)
			}
		}
	}
}

func (s *Scheduler) dmUserOnFailedMessage(msg *types.ScheduledMessage, postErr error) {
	channelInfo := s.channel.MakeChannelLink(s.channel.GetInfoOrUnknown(msg.ChannelID))
	message := formatter.FormatSchedulerFailure(channelInfo, postErr, msg.MessageContent)
	post := &model.Post{
		Message: message,
	}
	dmErr := s.api.Post.DM(s.botID, msg.UserID, post)
	if dmErr != nil {
		s.api.Log.Error("Failed to send failed message alert to user", "user", msg.UserID, "error", dmErr.Error())
	}
}
