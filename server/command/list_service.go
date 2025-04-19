package command

import (
	"fmt"
	"sort"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/formatter"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
)

type ListService struct {
	logger  ports.Logger
	store   store.Store
	channel ports.ChannelService
}

func NewListService(logger ports.Logger, store store.Store, channel ports.ChannelService) *ListService {
	return &ListService{
		logger:  logger,
		store:   store,
		channel: channel,
	}
}

func (l *ListService) Build(userID string) *model.CommandResponse {
	msgs, err := l.loadMessages(userID)
	if err != nil {
		return errorResponse(fmt.Sprintf("%s Error retrieving message list:  %v", constants.EmojiError, err))
	}
	if len(msgs) == 0 {
		return emptyResponse()
	}

	attachments := l.buildAttachments(msgs)
	return successResponse(attachments)
}

func (l *ListService) loadMessages(userID string) ([]*types.ScheduledMessage, error) {
	ids, err := l.store.ListUserMessageIDs(userID)
	if err != nil {
		return nil, err
	}

	var msgs []*types.ScheduledMessage
	for _, id := range ids {
		msg, err := l.store.GetScheduledMessage(id)
		if err != nil {
			continue
		}
		if msg.ID == "" {
			l.logger.Warn(fmt.Sprintf("Cleaning missing message %v from user index", id), "user_id", userID)
			_ = l.store.CleanupMessageFromUserIndex(userID, id)
			continue
		}
		msgs = append(msgs, msg)
	}

	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].PostAt.Before(msgs[j].PostAt)
	})

	return msgs, nil
}

func (l *ListService) buildAttachments(msgs []*types.ScheduledMessage) []*model.SlackAttachment {
	var attachments []*model.SlackAttachment
	channelCache := make(map[string]*ports.ChannelInfo)

	for _, m := range msgs {
		if _, ok := channelCache[m.ChannelID]; !ok {
			channelCache[m.ChannelID] = l.channel.GetInfoOrUnknown(m.ChannelID)
		}
		loc, _ := time.LoadLocation(m.Timezone)
		localTime := m.PostAt.In(loc)
		header := formatter.FormatListAttachmentHeader(
			localTime,
			l.channel.MakeChannelLink(channelCache[m.ChannelID]),
			m.MessageContent,
		)
		attachments = append(attachments, createAttachment(header, m.ID))
	}

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
