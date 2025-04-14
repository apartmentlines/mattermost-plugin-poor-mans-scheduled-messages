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
	case strings.HasPrefix(text, "help"):
		return h.scheduleHelp(), nil
	case strings.HasPrefix(text, "list"):
		return h.BuildEphemeralList(args), nil
	default:
		return h.handleSchedule(args, text), nil
	}
}

func (h *Handler) BuildEphemeralList(args *model.CommandArgs) *model.CommandResponse {
	h.client.Log.Debug("Building scheduled messages list", "user_id", args.UserId)
	ids, err := h.store.ListUserMessageIDs(args.UserId)
	if err != nil {
		message := fmt.Sprintf("❌ Error retrieving message list:  %v", err)
		h.client.Log.Error(message, "user_id", args.UserId)
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         message,
		}
	}
	idsLength := len(ids)
	if idsLength == 0 {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "You have no scheduled messages.",
		}
	}
	h.client.Log.Debug(fmt.Sprintf("Found %v scheduled message(s) in user index", idsLength), "user_id", args.UserId)

	var msgs []*types.ScheduledMessage
	for _, id := range ids {
		msg, err := h.store.GetScheduledMessage(id)
		if err == nil {
			// We don't have atomic operations for saving/deleting a message, so if it can't be found
			// clean up the user index as a failsafe.
			if msg.ID == "" {
				h.client.Log.Warn(fmt.Sprintf("Cleaning missing message %v from user index", id), "user_id", args.UserId)
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
			localTime.Format("Jan 2, 2006 3:04 PM"), m.MessageContent)
		attachments = append(attachments, createAttachment(header, m.ID))
	}

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         "### Scheduled Messages",
		Props: map[string]any{
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

func (h *Handler) UserDeleteMessage(userID string, msgID string) (*types.ScheduledMessage, error) {
	msg, err := h.store.GetScheduledMessage(msgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get scheduled message %s: %w", msgID, err)
	}
	if msg.UserID != userID {
		message := fmt.Sprintf("user %s attempted to delete message %s owned by %s", userID, msgID, msg.UserID)
		h.client.Log.Warn(message)
		return nil, errors.New(message)
	}
	err = h.store.DeleteScheduledMessage(userID, msgID)
	if err != nil {
		return nil, fmt.Errorf("failed to delete scheduled message %s: %w", msgID, err)
	}
	return msg, nil
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

func (h *Handler) scheduleHelp() *model.CommandResponse {
	// Use standard string concatenation to avoid issues with backticks in raw string literals.
	helpText := "\n### Schedule Messages Plugin Help\n\n" +
		"Use the `/schedule` command to send messages at a future time. Your configured Mattermost timezone is used for scheduling.\n\n" +
		"**Scheduling a Message:**\n\n" +
		"```\n/schedule at <time> [on <date>] message <your message text>\n```\n\n" +
		"*   `<time>`: Specify the time. Supported formats:\n" +
		"    *   `15:04` (military time)\n" +
		"    *   `3:04PM` (12 hour time)\n" +
		"    *   `3pm` (12 hour time shorthand)\n" +
		"*   `[on <date>]`: (Optional) Specify the date in `YYYY-MM-DD` format (e.g., `2025-12-31`).\n" +
		"    *   If omitted, defaults to today if the time is later than now, otherwise tomorrow.\n" +
		"*   `<message text>`: The content of the message.\n\n" +
		"**Examples:**\n\n" +
		"*   Schedule for 5 PM today/tomorrow:\n" +
		"    ```\n/schedule at 5:00PM message Remember the team meeting!\n```\n" +
		"*   Schedule for a specific date:\n" +
		"    ```\n/schedule at 9:30 on 2025-12-25 message Merry Christmas!\n```\n\n" +
		"**Listing Scheduled Messages:**\n\n" +
		"*   View your pending messages:\n" +
		"    ```\n/schedule list\n```\n" +
		"    *   This shows a list with options to delete each message.\n\n" +
		"**Getting Help:**\n\n" +
		"*   Show this help message:\n" +
		"    ```\n/schedule help\n```\n"

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         helpText,
	}
}

func (h *Handler) handleSchedule(args *model.CommandArgs, text string) *model.CommandResponse {
	h.client.Log.Debug("Trying to schedule message", "user_id", args.UserId, "text", text)
	if text == "" {
		return h.scheduleHelp()
	}
	parsed, parseInputErr := parseScheduleInput(text)
	if parseInputErr != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("Error: %v, Original input: `%v`", parseInputErr, text),
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

	if saveErr != nil {
		message := fmt.Sprintf("❌ Error scheduling message for %s (%s):  %v", schedTime.Format("Jan 2, 2006 3:04 PM"), tz, saveErr)
		h.client.Log.Error(message, "user_id", args.UserId)
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         message,
		}
	}
	message := fmt.Sprintf("✅ Scheduled message for %s (%s)", schedTime.Format("Jan 2, 2006 3:04 PM"), tz)
	h.client.Log.Info(message, "user_id", args.UserId)
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         message,
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
