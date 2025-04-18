package command

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/formatter"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/scheduler"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
)

type Handler struct {
	logger          ports.Logger
	slasher         ports.SlashCommandService
	user            ports.UserService
	store           store.Store
	scheduler       *scheduler.Scheduler
	channel         ports.ChannelService
	maxUserMessages int
	clock           ports.Clock
	helpText        string
}

func NewHandler(
	logger ports.Logger,
	slasher ports.SlashCommandService,
	user ports.UserService,
	store store.Store,
	sched *scheduler.Scheduler,
	channel ports.ChannelService,
	maxUserMessages int,
	clk ports.Clock,
	helpText string,
) *Handler {
	return &Handler{
		logger:          logger,
		slasher:         slasher,
		user:            user,
		store:           store,
		scheduler:       sched,
		channel:         channel,
		maxUserMessages: maxUserMessages,
		clock:           clk,
		helpText:        helpText,
	}
}

func (h *Handler) Register() error {
	err := h.slasher.Register(h.scheduleDefinition())
	if err != nil {
		h.logger.Error("Failed to register command", "error", err)
		return err
	}
	return nil
}

func (h *Handler) Execute(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	commandText := strings.TrimSpace(args.Command[len("/schedule"):])

	switch {
	case strings.HasPrefix(commandText, "help"):
		return h.scheduleHelp(), nil
	case strings.HasPrefix(commandText, "list"):
		return h.BuildEphemeralList(args), nil
	default:
		return h.handleSchedule(args, commandText), nil
	}
}

func (h *Handler) BuildEphemeralList(args *model.CommandArgs) *model.CommandResponse {
	h.logger.Debug("Building scheduled messages list", "user_id", args.UserId)
	ids, err := h.store.ListUserMessageIDs(args.UserId)
	if err != nil {
		message := fmt.Sprintf("❌ Error retrieving message list:  %v", err)
		h.logger.Error(message, "user_id", args.UserId)
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
	h.logger.Debug(fmt.Sprintf("Found %v scheduled message(s) in user index", idsLength), "user_id", args.UserId)

	var msgs []*types.ScheduledMessage
	channels := make(map[string]*ports.ChannelInfo)
	for _, id := range ids {
		msg, err := h.store.GetScheduledMessage(id)
		if err == nil {
			// We don't have atomic operations for saving/deleting a message, so if it can't be found
			// clean up the user index as a failsafe.
			if msg.ID == "" {
				h.logger.Warn(fmt.Sprintf("Cleaning missing message %v from user index", id), "user_id", args.UserId)
				_ = h.store.CleanupMessageFromUserIndex(msg.UserID, id)
			} else {
				if _, exists := channels[msg.ChannelID]; !exists {
					channels[msg.ChannelID] = h.channel.GetInfoOrUnknown(msg.ChannelID)
				}
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
		header := formatter.FormatListAttachmentHeader(localTime, h.channel.MakeChannelLink(channels[m.ChannelID]), m.MessageContent)
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
	user, err := h.user.Get(userID)
	if err != nil {
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
		h.logger.Warn(message)
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
		AutoCompleteHint: "[subcommand]",
		AutocompleteData: h.getScheduleAutocompleteData(),
		DisplayName:      "Schedule",
		Description:      "Send messages at a future time.",
	}
}

func (h *Handler) getScheduleAutocompleteData() *model.AutocompleteData {
	schedule := model.NewAutocompleteData("schedule", "[subcommand]", "Schedule messages")
	at := model.NewAutocompleteData("at", "<time> [on <date>] message <text>", "Schedule a new message")
	at.AddTextArgument("Time", "Time to send the message, e.g. 3:15PM, 3pm", "")
	at.AddTextArgument("Date", "(Optional) Date to send the message, e.g. 2026-01-01", "")
	at.AddTextArgument("Message", "The message content", "")
	schedule.AddCommand(at)
	list := model.NewAutocompleteData("list", "", "List your scheduled messages")
	schedule.AddCommand(list)
	help := model.NewAutocompleteData("help", "", "Show help text")
	schedule.AddCommand(help)
	return schedule
}

func (h *Handler) scheduleHelp() *model.CommandResponse {
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         h.helpText,
	}
}

func (h *Handler) checkMaxUserMessages(userID string) error {
	ids, userIndexErr := h.store.ListUserMessageIDs(userID)
	if userIndexErr != nil {
		return userIndexErr
	}
	if len(ids) >= h.maxUserMessages {
		return fmt.Errorf("you cannot schedule more than %d messages", h.maxUserMessages)
	}
	return nil
}

func (h *Handler) handleSchedule(args *model.CommandArgs, text string) *model.CommandResponse {
	h.logger.Debug("Trying to schedule message", "user_id", args.UserId, "text", text)
	maxMessagesErr := h.checkMaxUserMessages(args.UserId)
	if maxMessagesErr != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("❌ Error scheduling message: %v", maxMessagesErr),
		}
	}
	if text == "" {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "Trying to schedule a message? Use `/schedule help` for instructions.",
		}
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
	now := h.clock.Now().In(loc)

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
	channelInfo := h.channel.MakeChannelLink(h.channel.GetInfoOrUnknown(args.ChannelId))

	if saveErr != nil {
		message := formatter.FormatScheduleError(schedTime, tz, channelInfo, saveErr)
		h.logger.Error(message, "user_id", args.UserId)
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         message,
		}
	}
	message := formatter.FormatScheduleSuccess(schedTime, tz, channelInfo)
	h.logger.Info(message, "user_id", args.UserId)
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
