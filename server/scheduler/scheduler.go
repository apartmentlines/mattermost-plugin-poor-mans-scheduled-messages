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
)

type ChannelLinker interface {
	GetInfoOrUnknown(channelID string) *channel.Info
	MakeChannelLink(info *channel.Info) string
}

type Poster interface {
	CreatePost(post *model.Post) error
	DM(botID, userID string, post *model.Post) error
}

type tickerFactory func(d time.Duration) *time.Ticker

type Scheduler struct {
	poster    Poster
	logger    types.Logger
	store     store.Store
	linker    ChannelLinker
	botID     string
	clock     clock.Clock
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	newTicker tickerFactory
}

func New(poster Poster, logger types.Logger, store store.Store, linker ChannelLinker, botID string, clk clock.Clock) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		poster:    poster,
		logger:    logger,
		store:     store,
		linker:    linker,
		botID:     botID,
		clock:     clk,
		ctx:       ctx,
		cancel:    cancel,
		newTicker: time.NewTicker,
	}
}

func (s *Scheduler) Start() {
	ticker := s.newTicker(1 * time.Minute)
	defer ticker.Stop()
	s.run(ticker.C)
}

func (s *Scheduler) Stop() {
	s.cancel()
}

func (s *Scheduler) run(tick <-chan time.Time) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-tick:
			s.processDueMessages()
		}
	}
}

func (s *Scheduler) processDueMessages() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock.Now().UTC().Unix()
	ids, err := s.dueIDs()
	if err != nil {
		s.logger.Error("failed to list scheduled IDs", "err", err.Error())
		return
	}

	for id, ts := range ids {
		if ts > now {
			continue
		}
		msg, ok := s.loadMessage(id)
		if !ok {
			continue
		}
		s.handleDueMessage(msg)
	}
}

func (s *Scheduler) dueIDs() (map[string]int64, error) {
	return s.store.ListAllScheduledIDs()
}

func (s *Scheduler) loadMessage(id string) (*types.ScheduledMessage, bool) {
	msg, err := s.store.GetScheduledMessage(id)
	if err != nil {
		s.logger.Warn("unable to load scheduled message", "id", id, "err", err.Error())
		return nil, false
	}
	return msg, true
}

func (s *Scheduler) handleDueMessage(msg *types.ScheduledMessage) {
	if s.deleteSchedule(msg) != nil {
		return
	}
	if err := s.postMessage(msg); err != nil {
		s.dmUserOnFailedMessage(msg, err)
	} else {
		s.logger.Info("posted scheduled message", "id", msg.ID)
	}
}

func (s *Scheduler) deleteSchedule(msg *types.ScheduledMessage) error {
	err := s.store.DeleteScheduledMessage(msg.UserID, msg.ID)
	if err != nil {
		s.logger.Error("failed to delete scheduled message from store", "id", msg.ID, "err", err.Error())
	}
	return err
}

func (s *Scheduler) postMessage(msg *types.ScheduledMessage) error {
	post := &model.Post{
		ChannelId: msg.ChannelID,
		Message:   msg.MessageContent,
		UserId:    msg.UserID,
	}
	postErr := s.poster.CreatePost(post)
	if postErr != nil {
		s.logger.Error("failed to post scheduled message", "id", msg.ID, "err", postErr.Error())
	}
	return postErr
}

func (s *Scheduler) dmUserOnFailedMessage(msg *types.ScheduledMessage, postErr error) {
	channelInfo := s.linker.MakeChannelLink(s.linker.GetInfoOrUnknown(msg.ChannelID))
	message := formatter.FormatSchedulerFailure(channelInfo, postErr, msg.MessageContent)
	post := &model.Post{
		Message: message,
	}
	dmErr := s.poster.DM(s.botID, msg.UserID, post)
	if dmErr != nil {
		s.logger.Error("Failed to send failed message alert to user", "user", msg.UserID, "error", dmErr.Error())
	}
}
