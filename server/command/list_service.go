package command

import (
	"fmt"
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
	logger  ports.Logger
	store   ports.Store
	channel ports.ChannelService
}

// NewListService constructs a ListService.
func NewListService(logger ports.Logger, store ports.Store, channel ports.ChannelService) *ListService {
	logger.Debug("Creating new ListService")
	return &ListService{
		logger:  logger,
		store:   store,
		channel: channel,
	}
}

// Build creates a scheduled message list response.
func (l *ListService) Build(userID string) *model.CommandResponse {
	l.logger.Info("Building scheduled message list for user", "user_id", userID)
	msgs, err := l.loadMessages(userID)
	if err != nil {
		l.logger.Error("Failed to load messages for user", "user_id", userID, "error", err)
		return errorResponse(fmt.Sprintf("%s Error retrieving message list: %v", constants.EmojiError, err))
	}
	if len(msgs) == 0 {
		l.logger.Info("User has no scheduled messages", "user_id", userID)
		return emptyResponse()
	}

	l.logger.Debug("Successfully loaded messages, building attachments", "user_id", userID, "count", len(msgs))
	attachments := l.buildAttachments(msgs)
	l.logger.Debug("Successfully built attachments for message list", "user_id", userID, "count", len(attachments))
	return successResponse(attachments)
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
			// Log the error but continue trying to load other messages
			l.logger.Error("Failed to get scheduled message details", "user_id", userID, "message_id", id, "error", err)
			continue
		}
		if msg == nil || msg.ID == "" {
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

func (l *ListService) buildAttachments(msgs []*types.ScheduledMessage) []*model.SlackAttachment {
	l.logger.Debug("Building attachments for scheduled messages", "count", len(msgs))
	attachments := []*model.SlackAttachment{}
	channelCache := make(map[string]*ports.ChannelInfo)

	for _, m := range msgs {
		l.logger.Debug("Processing message for attachment", "message_id", m.ID, "channel_id", m.ChannelID)
		if _, ok := channelCache[m.ChannelID]; !ok {
			l.logger.Debug("Channel info not in cache, fetching", "channel_id", m.ChannelID)
			channelCache[m.ChannelID] = l.channel.GetInfoOrUnknown(m.ChannelID)
		} else {
			l.logger.Debug("Channel info found in cache", "channel_id", m.ChannelID)
		}
		loc, _ := time.LoadLocation(m.Timezone)
		localTime := m.PostAt.In(loc)
		header := formatter.FormatListAttachmentHeader(
			localTime,
			l.channel.MakeChannelLink(channelCache[m.ChannelID]),
			m.MessageContent,
		)
		attachments = append(attachments, createAttachment(header, m.ID))
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

func emptyResponse() *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         constants.EmptyListMessage,
	}
}

func successResponse(atts []*model.SlackAttachment) *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         constants.ListHeader,
		Props: map[string]any{
			"attachments": atts,
		},
	}
}

func createAttachment(text string, messageID string) *model.SlackAttachment {
	return &model.SlackAttachment{
		Text: text,
		Actions: []*model.PostAction{
			{
				Id:   "send",
				Name: "Send",
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
				Name:  "Delete",
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
