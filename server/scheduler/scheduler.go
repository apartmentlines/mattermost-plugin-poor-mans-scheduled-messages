package scheduler

import (
	"context"
	"sync"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/formatter"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
)

type tickerFactory func(d time.Duration) *time.Ticker

type Scheduler struct {
	logger ports.Logger
	poster ports.PostService
	store  ports.Store
	linker ports.ChannelService
	botID  string
	clock  ports.Clock
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
}

func New(logger ports.Logger, poster ports.PostService, store ports.Store, linker ports.ChannelService, botID string, clk ports.Clock) *Scheduler {
	logger.Debug("Creating new scheduler instance")
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		logger: logger,
		poster: poster,
		store:  store,
		linker: linker,
		botID:  botID,
		clock:  clk,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Scheduler) Start() {
	s.logger.Info("Scheduler starting")
	go s.run()
}

func (s *Scheduler) Stop() {
	s.logger.Info("Scheduler stopping")
	s.cancel()
	s.logger.Info("Scheduler stopped")
}

func (s *Scheduler) run() {
	s.logger.Debug("Scheduler run loop started")
	defer s.logger.Info("Scheduler run loop exited")

	for {
		now := s.clock.Now()
		nextMinute := now.Truncate(time.Minute).Add(time.Minute)
		duration := nextMinute.Sub(now)

		if duration <= 0 {
			duration += time.Minute
			nextMinute = nextMinute.Add(time.Minute)
		}

		s.logger.Debug("Scheduler waiting for next minute", "wait_duration", duration, "target_time", nextMinute)
		timer := time.NewTimer(duration)

		select {
		case <-s.ctx.Done():
			s.logger.Debug("Scheduler context done, stopping timer and exiting run loop")
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			return
		case t := <-timer.C:
			s.logger.Debug("Scheduler received timer tick", "time", t)
			s.processDueMessages()
		}
	}
}

func (s *Scheduler) processDueMessages() {
	s.logger.Debug("Processing due messages")
	s.mu.Lock()
	s.logger.Debug("Acquired scheduler lock")
	defer func() {
		s.mu.Unlock()
		s.logger.Debug("Released scheduler lock")
	}()

	now := s.clock.Now().UTC()
	nowUnix := now.Unix()
	s.logger.Debug("Current time for due check", "time_utc", now, "time_unix", nowUnix)

	ids, err := s.dueIDs()
	if err != nil {
		s.logger.Error("Failed to list scheduled message IDs", "error", err)
		return
	}
	s.logger.Debug("Retrieved scheduled message IDs", "count", len(ids))

	processedCount := 0
	skippedCount := 0
	loadFailedCount := 0
	for id, ts := range ids {
		if ts > nowUnix {
			// s.logger.Debug("Skipping message, not due yet", "message_id", id, "post_at_unix", ts, "now_unix", nowUnix)
			skippedCount++
			continue
		}
		s.logger.Debug("Message is due, loading", "message_id", id, "post_at_unix", ts, "now_unix", nowUnix)
		msg, ok := s.loadMessage(id)
		if !ok {
			loadFailedCount++
			continue
		}
		s.handleDueMessage(msg)
		processedCount++
	}
	s.logger.Debug("Finished processing potential messages", "processed", processedCount, "skipped_not_due", skippedCount, "load_failed", loadFailedCount, "total_candidates", len(ids))
}

func (s *Scheduler) dueIDs() (map[string]int64, error) {
	s.logger.Debug("Listing all scheduled message IDs from store")
	ids, err := s.store.ListAllScheduledIDs()
	if err == nil {
		s.logger.Debug("Successfully listed scheduled IDs", "count", len(ids))
	}
	return ids, err
}

func (s *Scheduler) loadMessage(id string) (*types.ScheduledMessage, bool) {
	s.logger.Debug("Loading scheduled message from store", "message_id", id)
	msg, err := s.store.GetScheduledMessage(id)
	if err != nil {
		s.logger.Warn("Unable to load scheduled message", "message_id", id, "error", err)
		return nil, false
	}
	s.logger.Debug("Successfully loaded scheduled message", "message_id", id, "user_id", msg.UserID, "channel_id", msg.ChannelID, "post_at", msg.PostAt)
	return msg, true
}

func (s *Scheduler) handleDueMessage(msg *types.ScheduledMessage) {
	s.logger.Debug("Handling due message", "message_id", msg.ID, "user_id", msg.UserID, "channel_id", msg.ChannelID)
	if s.deleteSchedule(msg) != nil {
		s.logger.Error("Halting processing for message due to delete failure", "message_id", msg.ID)
		return
	}
	if err := s.postMessage(msg); err != nil {
		s.logger.Warn("Message posting failed, attempting to DM user", "message_id", msg.ID, "user_id", msg.UserID, "error", err)
		s.dmUserOnFailedMessage(msg, err)
	} else {
		s.logger.Info("Successfully posted scheduled message", "message_id", msg.ID, "user_id", msg.UserID, "channel_id", msg.ChannelID, "post_at", msg.PostAt)
	}
}

func (s *Scheduler) deleteSchedule(msg *types.ScheduledMessage) error {
	s.logger.Debug("Deleting scheduled message from store", "message_id", msg.ID, "user_id", msg.UserID)
	err := s.store.DeleteScheduledMessage(msg.UserID, msg.ID)
	if err != nil {
		s.logger.Error("Failed to delete scheduled message from store", "message_id", msg.ID, "user_id", msg.UserID, "error", err)
	}
	s.logger.Debug("Successfully deleted scheduled message", "message_id", msg.ID, "user_id", msg.UserID)
	return err
}

func (s *Scheduler) postMessage(msg *types.ScheduledMessage) error {
	s.logger.Debug("Attempting to post scheduled message", "message_id", msg.ID, "user_id", msg.UserID, "channel_id", msg.ChannelID)
	post := &model.Post{
		ChannelId: msg.ChannelID,
		Message:   msg.MessageContent,
		UserId:    msg.UserID,
	}
	postErr := s.poster.CreatePost(post)
	if postErr != nil {
		s.logger.Error("Failed to post scheduled message via PostService", "message_id", msg.ID, "user_id", msg.UserID, "channel_id", msg.ChannelID, "error", postErr)
	}
	s.logger.Debug("Successfully created post via PostService", "message_id", msg.ID, "user_id", msg.UserID, "channel_id", msg.ChannelID)
	return postErr
}

func (s *Scheduler) dmUserOnFailedMessage(msg *types.ScheduledMessage, postErr error) {
	s.logger.Debug("Attempting to DM user about failed message", "message_id", msg.ID, "user_id", msg.UserID, "original_channel_id", msg.ChannelID, "post_error", postErr)
	channelInfo := s.linker.MakeChannelLink(s.linker.GetInfoOrUnknown(msg.ChannelID))
	message := formatter.FormatSchedulerFailure(channelInfo, postErr, msg.MessageContent)
	post := &model.Post{
		Message: message,
	}
	dmErr := s.poster.DM(s.botID, msg.UserID, post)
	if dmErr != nil {
		s.logger.Error("Failed to send DM alert to user about failed scheduled message", "message_id", msg.ID, "user_id", msg.UserID, "dm_error", dmErr, "original_post_error", postErr)
	} else {
		s.logger.Debug("Successfully sent DM alert to user", "message_id", msg.ID, "user_id", msg.UserID)
	}
}
