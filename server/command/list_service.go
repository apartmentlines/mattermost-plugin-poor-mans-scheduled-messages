package command

import (
	"errors"
	"sort"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/formatter"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
)

// ListService builds scheduled message lists.
type ListService struct {
	presentation ports.DisplayPreferences
	logger       ports.Logger
	store        ports.Store
	channel      ports.ChannelService
}

// NewListService constructs a ListService.
func NewListService(logger ports.Logger, store ports.Store, channel ports.ChannelService, presentation ports.DisplayPreferences) *ListService {
	logger.Debug("Creating new ListService")
	return &ListService{
		presentation: presentation,
		logger:       logger,
		store:        store,
		channel:      channel,
	}
}

// Build creates a scheduled message list response.
func (l *ListService) Build(userID string) *model.CommandResponse {
	r := formatter.ForUser(l.presentation, userID)
	l.logger.Info("Building scheduled message list for user", "user_id", userID)
	msgs, err := l.loadMessages(userID)
	if err != nil {
		l.logger.Error("Failed to load messages for user", "user_id", userID, "error", err)
		return errorResponse(r.Text("list_error", constants.EmojiError, err))
	}
	if len(msgs) == 0 {
		l.logger.Info("User has no scheduled messages", "user_id", userID)
		return emptyResponse(r)
	}

	l.logger.Debug("Successfully loaded messages, building attachments", "user_id", userID, "count", len(msgs))
	attachments := l.buildAttachments(msgs, r)
	l.logger.Debug("Successfully built attachments for message list", "user_id", userID, "count", len(attachments))
	return successResponse(attachments, r)
}

func (l *ListService) loadMessages(userID string) ([]*types.ScheduledMessage, error) {
	l.logger.Debug("Loading scheduled message IDs for user", "user_id", userID)
	ids, err := l.store.ListUserMessageIDs(userID)
	if err != nil {
		l.logger.Error("Failed to list user message IDs", "user_id", userID, "error", err)
		return nil, err
	}
	l.logger.Debug("Found message IDs for user", "user_id", userID, "count", len(ids))

	msgs := []*types.ScheduledMessage{}
	for _, id := range ids {
		l.logger.Debug("Loading scheduled message details", "user_id", userID, "message_id", id)
		msg, err := l.store.GetScheduledMessage(id)
		if err != nil {
			if errors.Is(err, types.ErrMessageNotFound) {
				_ = l.store.CleanupMessageFromUserIndex(userID, id)
			}
			// Log the error but continue trying to load other messages
			l.logger.Error("Failed to get scheduled message details", "user_id", userID, "message_id", id, "error", err)
			continue
		}
		if msg == nil || msg.ID == "" || msg.Terminal() {
			l.logger.Warn("Scheduled message referenced in user index not found, cleaning up", "user_id", userID, "message_id", id)
			// Attempt cleanup, log if cleanup fails but continue processing
			if cleanupErr := l.store.CleanupMessageFromUserIndex(userID, id); cleanupErr != nil {
				l.logger.Error("Failed to cleanup missing message from user index", "user_id", userID, "message_id", id, "error", cleanupErr)
			}
			continue
		}
		l.logger.Debug("Successfully loaded scheduled message", "user_id", userID, "message_id", msg.ID)
		msgs = append(msgs, msg)
	}

	l.logger.Debug("Sorting loaded messages by post time", "user_id", userID, "count", len(msgs))
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].PostAt.Before(msgs[j].PostAt)
	})

	l.logger.Debug("Finished loading and sorting messages for user", "user_id", userID, "count", len(msgs))
	return msgs, nil
}

func (l *ListService) buildAttachments(msgs []*types.ScheduledMessage, r formatter.Renderer) []*model.MessageAttachment {
	l.logger.Debug("Building attachments for scheduled messages", "count", len(msgs))
	attachments := []*model.MessageAttachment{}
	channelCache := make(map[string]*ports.ChannelInfo)

	for _, m := range msgs {
		l.logger.Debug("Processing message for attachment", "message_id", m.ID, "channel_id", m.ChannelID)
		if _, ok := channelCache[m.ChannelID]; !ok {
			l.logger.Debug("Channel info not in cache, fetching", "channel_id", m.ChannelID)
			channelCache[m.ChannelID] = l.channel.GetInfoOrUnknown(m.ChannelID)
		} else {
			l.logger.Debug("Channel info found in cache", "channel_id", m.ChannelID)
		}
		loc, err := time.LoadLocation(m.Timezone)
		if err != nil {
			loc = time.UTC
		}
		localTime := m.PostAt.In(loc)
		header := r.FormatListAttachmentHeader(
			localTime,
			l.channel.MakeChannelLink(channelCache[m.ChannelID], r.DisplaySettings),
			m.MessageContent,
			m.RootID != "",
		)
		if m.State != "" {
			header += "\n\n" + r.Text("delivery_status", m.State, m.Attempts, types.MaxDeliveryAttempts)
			if m.LastReason != "" {
				header += "\n" + r.Text("delivery_reason", m.LastReason)
			}
			if !m.NextAttemptAt.IsZero() {
				header += "\n" + r.Text("next_check", r.DateTime(m.NextAttemptAt.In(loc)))
			}
		}
		attachment := createAttachment(header, m.ID, r)
		switch m.DeliveryState() {
		case types.StateSending, types.StateReconciling:
			attachment.Actions = nil
		case types.StateUncertain:
			attachment.Actions = attachment.Actions[:1]
			attachment.Actions[0].Name = r.Text("check_delivery")
		case types.StateCreating, types.StateProcessing:
			attachment.Actions = attachment.Actions[1:]
		}
		attachments = append(attachments, attachment)
		l.logger.Debug("Created attachment for message", "message_id", m.ID)
	}

	l.logger.Debug("Finished building all attachments", "count", len(attachments))
	return attachments
}

func errorResponse(txt string) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         txt,
	}
}

func emptyResponse(r formatter.Renderer) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         r.Text("empty_list"),
	}
}

func successResponse(atts []*model.MessageAttachment, r formatter.Renderer) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         r.Text("list_header"),
		Props: map[string]any{
			"attachments": atts,
		},
	}
}

func createAttachment(text string, messageID string, r formatter.Renderer) *model.MessageAttachment {
	return &model.MessageAttachment{
		Text: text,
		Actions: []*model.PostAction{
			{
				Id:   "send",
				Name: r.Text("send"),
				Integration: &model.PostActionIntegration{
					URL: "/plugins/com.mattermost.plugin-poor-mans-scheduled-messages/api/v1/send",
					Context: map[string]any{
						"action": "send",
						"id":     messageID,
					},
				},
			},
			{
				Id:    "delete",
				Name:  r.Text("delete"),
				Style: "danger",
				Integration: &model.PostActionIntegration{
					URL: "/plugins/com.mattermost.plugin-poor-mans-scheduled-messages/api/v1/delete",
					Context: map[string]any{
						"action": "delete",
						"id":     messageID,
					},
				},
			},
		},
	}
}
