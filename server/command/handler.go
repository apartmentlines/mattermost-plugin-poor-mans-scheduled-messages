package command

import (
	"errors"
	"fmt"
	"strings"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
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
	commandText := strings.TrimSpace(args.Command[len("/"+constants.CommandTrigger):])

	switch {
	case strings.HasPrefix(commandText, constants.SubcommandHelp):
		return h.scheduleHelp(), nil
	case strings.HasPrefix(commandText, constants.SubcommandList):
		return h.BuildEphemeralList(args), nil
	default:
		// Assume it's a schedule request if not help or list
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
		Trigger:          constants.CommandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: constants.AutocompleteDesc,
		AutoCompleteHint: constants.AutocompleteHint,
		AutocompleteData: h.getScheduleAutocompleteData(),
		DisplayName:      constants.CommandDisplayName,
		Description:      constants.CommandDescription,
	}
}

func (h *Handler) getScheduleAutocompleteData() *model.AutocompleteData {
	schedule := model.NewAutocompleteData(constants.CommandTrigger, constants.AutocompleteHint, constants.AutocompleteDesc)

	at := model.NewAutocompleteData(constants.SubcommandAt, constants.AutocompleteAtHint, constants.AutocompleteAtDesc)
	at.AddTextArgument(constants.AutocompleteAtArgTimeName, constants.AutocompleteAtArgTimeHint, "")
	at.AddTextArgument(constants.AutocompleteAtArgDateName, constants.AutocompleteAtArgDateHint, "")
	at.AddTextArgument(constants.AutocompleteAtArgMsgName, constants.AutocompleteAtArgMsgHint, "")
	schedule.AddCommand(at)

	list := model.NewAutocompleteData(constants.SubcommandList, constants.AutocompleteListHint, constants.AutocompleteListDesc)
	schedule.AddCommand(list)

	help := model.NewAutocompleteData(constants.SubcommandHelp, constants.AutocompleteHelpHint, constants.AutocompleteHelpDesc)
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
