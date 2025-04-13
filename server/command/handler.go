package command

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/scheduler"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type Handler struct {
	client    *pluginapi.Client
	store     store.Store
	scheduler *scheduler.Scheduler
}

func NewHandler(client *pluginapi.Client, store store.Store, sched *scheduler.Scheduler) *Handler {
	return &Handler{
		client:    client,
		store:     store,
		scheduler: sched,
	}
}

func (h *Handler) Register() error {
	err := h.client.SlashCommand.Register(h.scheduleDefinition())
	if err != nil {
		h.client.Log.Error("Failed to register command", "error", err)
		return err
	}
	return nil
}

func (h *Handler) Execute(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	text := strings.TrimSpace(args.Command[len("/schedule"):])

	switch {
	case strings.HasPrefix(text, "list"):
		return h.BuildEphemeralList(args), nil
	default:
		return h.handleSchedule(args, text), nil
	}
}

func (h *Handler) BuildEphemeralList(args *model.CommandArgs) *model.CommandResponse {
	ids, err := h.store.ListUserMessageIDs(args.UserId)
	if err != nil || len(ids) == 0 {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "You have no scheduled messages.",
		}
	}

	var msgs []*types.ScheduledMessage
	for _, id := range ids {
		msg, err := h.store.GetScheduledMessage(id)
		if err == nil {
			// We don't have atomic operations for saving/deleting a message, so if it can't be found
			// clean up the user index as a failsafe.
			if msg.ID == "" {
				_ = h.store.CleanupMessageFromUserIndex(msg.UserID, id)
			} else {
				msgs = append(msgs, msg)
			}
		}
	}
	sort.Slice(msgs, func(i, j int) bool {
		return msgs[i].PostAt.Before(msgs[j].PostAt)
	})

	var attachments []*model.SlackAttachment
	for _, m := range msgs {
		loc, _ := time.LoadLocation(m.Timezone)
		localTime := m.PostAt.In(loc)
		header := fmt.Sprintf("### %s\n%s",
			localTime.Format("2006-01-02 15:04"), m.MessageContent)
		attachments = append(attachments, createAttachment(header, m.ID))
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         "### Scheduled Messages",
		Props: map[string]interface{}{
			"attachments": attachments,
		},
	}
}

func (h *Handler) GetUserTimezone(userID string) string {
	user, appErr := h.client.User.Get(userID)
	if appErr != nil {
		return "UTC"
	}
	automaticTimezone, aok := user.Timezone["automaticTimezone"]
	useAutomaticTimezone, uok := user.Timezone["useAutomaticTimezone"]
	manualTimezone, mok := user.Timezone["manualTimezone"]
	if aok && uok && automaticTimezone != "" && useAutomaticTimezone == "true" {
		return automaticTimezone
	} else if mok && manualTimezone != "" {
		return manualTimezone
	}
	return "UTC"
}

func (h *Handler) UserDeleteMessage(userID string, msgID string) error {
	msg, err := h.store.GetScheduledMessage(msgID)
	if err != nil {
		return err
	}
	if msg.UserID != userID {
		message := fmt.Sprintf("User %s does not own message", userID)
		h.client.Log.Warn(message)
		return errors.New(message)
	}
	err = h.store.DeleteScheduledMessage(userID, msgID)
	if err != nil {
		return err
	}
	return nil
}

func (h *Handler) scheduleDefinition() *model.Command {
	return &model.Command{
		Trigger:          "schedule",
		AutoComplete:     true,
		AutoCompleteDesc: "Schedule messages to be sent later",
		AutoCompleteHint: "at <time> [on <date>] message <text> | list",
		DisplayName:      "Schedule",
		Description:      "Send messages at a future time.",
	}
}

func (h *Handler) handleSchedule(args *model.CommandArgs, text string) *model.CommandResponse {
	parsed, parseInputErr := parseScheduleInput(text)
	if parseInputErr != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("Error: %v", parseInputErr),
		}
	}

	tz := h.GetUserTimezone(args.UserId)
	loc, _ := time.LoadLocation(tz)
	now := time.Now().In(loc)

	schedTime, resolveErr := resolveScheduledTime(parsed.TimeStr, parsed.DateStr, now, loc)
	if resolveErr != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("Error parsing time/date: %v", resolveErr),
		}
	}

	id := h.store.GenerateMessageID()
	msg := &types.ScheduledMessage{
		ID:             id,
		UserID:         args.UserId,
		ChannelID:      args.ChannelId,
		PostAt:         schedTime.UTC(),
		MessageContent: parsed.Message,
		Timezone:       tz,
	}

	saveErr := h.store.SaveScheduledMessage(args.UserId, msg)

	if saveErr == nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("✅ Scheduled message for %s (%s)", schedTime.Format("2006-01-02 15:04"), tz),
		}
	}
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         fmt.Sprintf("❌ Error scheduling message for %s (%s):  %v", schedTime.Format("2006-01-02 15:04"), tz, saveErr),
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
					Context: map[string]interface{}{
						"action": "delete",
						"id":     messageID,
					},
				},
			},
		},
	}
}
