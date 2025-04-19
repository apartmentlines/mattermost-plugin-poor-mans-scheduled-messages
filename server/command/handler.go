package command

import (
	"fmt"
	"strings"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
)

type Handler struct {
	logger          ports.Logger
	slasher         ports.SlashCommandService
	user            ports.UserService
	store           ports.Store
	channel         ports.ChannelService
	listService     ports.ListService
	scheduleService ports.ScheduleService
	helpText        string
}

func NewHandler(
	logger ports.Logger,
	slasher ports.SlashCommandService,
	user ports.UserService,
	store ports.Store,
	channel ports.ChannelService,
	listSvc ports.ListService,
	scheduleSvc ports.ScheduleService,
	helpText string,
) *Handler {
	logger.Debug("Creating new command Handler")
	return &Handler{
		logger:          logger,
		slasher:         slasher,
		user:            user,
		store:           store,
		channel:         channel,
		listService:     listSvc,
		scheduleService: scheduleSvc,
		helpText:        helpText,
	}
}

func (h *Handler) Register() error {
	h.logger.Debug("Registering slash command")
	err := h.slasher.Register(h.scheduleDefinition())
	if err != nil {
		h.logger.Error("Failed to register slash command", "trigger", constants.CommandTrigger, "error", err)
		return err
	}
	return nil
}

func (h *Handler) Execute(args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	h.logger.Debug("Executing command", "user_id", args.UserId, "channel_id", args.ChannelId, "command", args.Command)
	commandText := strings.TrimSpace(args.Command[len("/"+constants.CommandTrigger):])

	switch {
	case strings.HasPrefix(commandText, constants.SubcommandHelp):
		h.logger.Debug("Handling help subcommand", "user_id", args.UserId)
		return h.scheduleHelp(), nil
	case strings.HasPrefix(commandText, constants.SubcommandList):
		h.logger.Debug("Handling list subcommand", "user_id", args.UserId)
		return h.BuildEphemeralList(args), nil
	default:
		h.logger.Debug("Handling schedule subcommand", "user_id", args.UserId, "command_text", commandText)
		return h.handleSchedule(args, commandText), nil
	}
}

func (h *Handler) BuildEphemeralList(args *model.CommandArgs) *model.CommandResponse {
	h.logger.Debug("Building ephemeral list response", "user_id", args.UserId)
	return h.listService.Build(args.UserId)
}

func (h *Handler) UserDeleteMessage(userID string, msgID string) (*types.ScheduledMessage, error) {
	h.logger.Debug("Attempting to delete message", "user_id", userID, "message_id", msgID)
	msg, err := h.store.GetScheduledMessage(msgID)
	if err != nil {
		h.logger.Error("Failed to get scheduled message for deletion", "message_id", msgID, "error", err)
		return nil, fmt.Errorf("failed to get scheduled message %s: %w", msgID, err)
	}
	if msg.UserID != userID {
		h.logger.Warn("User attempted to delete message owned by another user", "requesting_user_id", userID, "message_id", msgID, "owner_user_id", msg.UserID)
		return nil, fmt.Errorf("user %s attempted to delete message %s owned by %s", userID, msgID, msg.UserID)
	}
	err = h.store.DeleteScheduledMessage(userID, msgID)
	if err != nil {
		h.logger.Error("Failed to delete scheduled message from store", "user_id", userID, "message_id", msgID, "error", err)
		return nil, fmt.Errorf("failed to delete scheduled message %s: %w", msgID, err)
	}
	h.logger.Info("Successfully deleted scheduled message", "user_id", userID, "message_id", msgID)
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
	h.logger.Debug("Generating help response")
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         h.helpText,
	}
}

func (h *Handler) handleSchedule(args *model.CommandArgs, text string) *model.CommandResponse {
	h.logger.Debug("Building schedule response", "user_id", args.UserId, "command_text", text)
	return h.scheduleService.Build(args, text)
}
