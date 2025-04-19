package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/formatter"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
)

type ScheduleService struct {
	logger          ports.Logger
	userAPI         ports.UserService
	store           store.Store
	channel         ports.ChannelService
	clock           ports.Clock
	maxUserMessages int
}

func NewScheduleService(
	logger ports.Logger,
	userAPI ports.UserService,
	store store.Store,
	channel ports.ChannelService,
	clk ports.Clock,
	maxUserMessages int,
) *ScheduleService {
	return &ScheduleService{
		logger:          logger,
		userAPI:         userAPI,
		store:           store,
		channel:         channel,
		clock:           clk,
		maxUserMessages: maxUserMessages,
	}
}

func (s *ScheduleService) Build(args *model.CommandArgs, text string) *model.CommandResponse {
	s.logger.Debug("Trying to schedule message", "user_id", args.UserId, "text", text)
	if resp := s.validateRequest(args.UserId, text); resp != nil {
		return resp
	}
	msg, tz, err := s.prepareSchedule(args.UserId, args.ChannelId, text)
	if err != nil {
		return s.errorResponse(fmt.Sprintf("Error: %v, Original input: `%v`", err, text))
	}
	if err := s.persist(args.UserId, msg); err != nil {
		channelLink := s.channel.MakeChannelLink(s.channel.GetInfoOrUnknown(args.ChannelId))
		formatted := formatter.FormatScheduleError(msg.PostAt, tz, channelLink, err)
		return s.errorResponse(formatted)
	}
	return s.successResponse(msg, tz, args.ChannelId)
}

func (s *ScheduleService) checkMaxUserMessages(userID string) error {
	ids, err := s.store.ListUserMessageIDs(userID)
	if err != nil {
		return err
	}
	if len(ids) >= s.maxUserMessages {
		return fmt.Errorf("you cannot schedule more than %d messages", s.maxUserMessages)
	}
	return nil
}

func (s *ScheduleService) checkMaxMessageBytes(text string) error {
	if len(text) > constants.MaxMessageBytes {
		kb := float64(constants.MaxMessageBytes) / 1024
		return fmt.Errorf("you cannot schedule a message longer than %v", fmt.Sprintf("%.2f KB", kb))
	}
	return nil
}

func (s *ScheduleService) getUserTimezone(userID string) string {
	user, err := s.userAPI.Get(userID)
	if err != nil {
		return constants.DefaultTimezone
	}
	automaticTimezone, aok := user.Timezone["automaticTimezone"]
	useAutomaticTimezone, uok := user.Timezone["useAutomaticTimezone"]
	manualTimezone, mok := user.Timezone["manualTimezone"]
	if aok && uok && automaticTimezone != "" && useAutomaticTimezone == "true" {
		return automaticTimezone
	}
	if mok && manualTimezone != "" {
		return manualTimezone
	}
	return constants.DefaultTimezone
}

func (s *ScheduleService) validateRequest(userID, text string) *model.CommandResponse {
	if maxUserMessagesErr := s.checkMaxUserMessages(userID); maxUserMessagesErr != nil {
		return s.errorResponse(formatter.FormatScheduleValidationError(maxUserMessagesErr))
	}
	if maxMessageBytesErr := s.checkMaxMessageBytes(text); maxMessageBytesErr != nil {
		return s.errorResponse(formatter.FormatScheduleValidationError(maxMessageBytesErr))
	}
	if strings.TrimSpace(text) == "" {
		return s.errorResponse(formatter.FormatEmptyCommandError())
	}
	return nil
}

func (s *ScheduleService) persist(userID string, msg *types.ScheduledMessage) error {
	return s.store.SaveScheduledMessage(userID, msg)
}

func (s *ScheduleService) errorResponse(text string) *model.CommandResponse {
	s.logger.Error(text)
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}
}

func (s *ScheduleService) prepareSchedule(userID, channelID, text string) (*types.ScheduledMessage, string, error) {
	parsed, parseErr := parseScheduleInput(text)
	if parseErr != nil {
		return nil, "", parseErr
	}
	tz := s.getUserTimezone(userID)
	loc, _ := time.LoadLocation(tz)
	now := s.clock.Now().In(loc)
	schedTime, resolveErr := resolveScheduledTime(parsed.TimeStr, parsed.DateStr, now, loc)
	if resolveErr != nil {
		return nil, "", resolveErr
	}
	msg := &types.ScheduledMessage{
		ID:             s.store.GenerateMessageID(),
		UserID:         userID,
		ChannelID:      channelID,
		PostAt:         schedTime.UTC(),
		MessageContent: parsed.Message,
		Timezone:       tz,
	}
	return msg, tz, nil
}

func (s *ScheduleService) successResponse(msg *types.ScheduledMessage, tz, channelID string) *model.CommandResponse {
	channelLink := s.channel.MakeChannelLink(s.channel.GetInfoOrUnknown(channelID))
	text := formatter.FormatScheduleSuccess(msg.PostAt, tz, channelLink)
	s.logger.Info(text, "user_id", msg.UserID)
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}
}
