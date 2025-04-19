package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
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
	listService     *ListService
	scheduleService *ScheduleService
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
		listService:     NewListService(logger, store, channel),
		scheduleService: NewScheduleService(logger, user, store, channel, clk, maxUserMessages),
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
	return h.listService.Build(args.UserId)
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

func (h *Handler) handleSchedule(args *model.CommandArgs, text string) *model.CommandResponse {
	return h.scheduleService.Build(args, text)
}
