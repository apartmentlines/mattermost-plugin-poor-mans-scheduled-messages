package command

import (
	"fmt"
	"strings"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/formatter"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
)

type ScheduleService struct {
	logger          ports.Logger
	userAPI         ports.UserService
	store           ports.Store
	channel         ports.ChannelService
	clock           ports.Clock
	maxUserMessages int
}

func NewScheduleService(
	logger ports.Logger,
	userAPI ports.UserService,
	store ports.Store,
	channel ports.ChannelService,
	clk ports.Clock,
	maxUserMessages int,
) *ScheduleService {
	logger.Debug("Creating new ScheduleService")
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
	s.logger.Debug("Attempting to schedule message", "user_id", args.UserId, "channel_id", args.ChannelId, "text", text)

	s.logger.Debug("Validating schedule request", "user_id", args.UserId)
	if resp := s.validateRequest(args.UserId, text); resp != nil {
		s.logger.Error("Schedule request validation failed", "user_id", args.UserId, "reason", resp.Text)
		return resp
	}
	s.logger.Debug("Schedule request validated successfully", "user_id", args.UserId)

	s.logger.Debug("Preparing schedule details", "user_id", args.UserId, "channel_id", args.ChannelId)
	msg, loc, tz, err := s.prepareSchedule(args.UserId, args.ChannelId, text)
	if err != nil {
		errMsg := fmt.Sprintf("Error preparing schedule: %v, Original input: `%v`", err, text)
		s.logger.Error("Failed to prepare schedule", "user_id", args.UserId, "channel_id", args.ChannelId, "error", err, "original_text", text)
		return s.errorResponse(errMsg)
	}
	localTime := msg.PostAt.In(loc)
	s.logger.Debug("Schedule details prepared", "user_id", args.UserId, "message_id", msg.ID, "post_at", localTime, "timezone", tz)

	s.logger.Debug("Persisting scheduled message", "user_id", args.UserId, "message_id", msg.ID)
	if err := s.persist(args.UserId, msg); err != nil {
		channelLink := s.channel.MakeChannelLink(s.channel.GetInfoOrUnknown(args.ChannelId))
		formatted := formatter.FormatScheduleError(localTime, tz, channelLink, err)
		s.logger.Error("Failed to persist scheduled message", "user_id", args.UserId, "message_id", msg.ID, "error", err)
		return s.errorResponse(formatted)
	}
	s.logger.Info("Scheduled message persisted successfully", "user_id", args.UserId, "message_id", msg.ID)

	return s.successResponse(msg, localTime, tz, args.ChannelId)
}

func (s *ScheduleService) checkMaxUserMessages(userID string) error {
	s.logger.Debug("Checking max user messages limit", "user_id", userID, "limit", s.maxUserMessages)
	ids, err := s.store.ListUserMessageIDs(userID)
	if err != nil {
		s.logger.Error("Failed to list user message IDs for count check", "user_id", userID, "error", err)
		return fmt.Errorf("failed to check message count: %w", err)
	}
	count := len(ids)
	s.logger.Debug("Current user message count", "user_id", userID, "count", count)
	if count >= s.maxUserMessages {
		err := fmt.Errorf("cannot schedule more than %d messages (current: %d)", s.maxUserMessages, count)
		s.logger.Error("User message limit reached", "user_id", userID, "count", count, "limit", s.maxUserMessages)
		return err
	}
	s.logger.Debug("User is under message limit", "user_id", userID, "count", count, "limit", s.maxUserMessages)
	return nil
}

func (s *ScheduleService) checkMaxMessageBytes(text string) error {
	length := len(text)
	s.logger.Debug("Checking max message bytes", "length", length, "limit", constants.MaxMessageBytes)
	if length > constants.MaxMessageBytes {
		kb := float64(constants.MaxMessageBytes) / 1024
		userKb := float64(length) / 1024
		err := fmt.Errorf("message length %.2f KB exceeds limit %.2f KB", userKb, kb)
		s.logger.Error("Message length exceeds limit", "length", length, "limit", constants.MaxMessageBytes)
		return err
	}
	s.logger.Debug("Message length is within limit", "length", length, "limit", constants.MaxMessageBytes)
	return nil
}

func (s *ScheduleService) getUserTimezone(userID string) string {
	s.logger.Debug("Attempting to get user timezone", "user_id", userID)
	user, err := s.userAPI.Get(userID)
	if err != nil {
		s.logger.Warn("Failed to get user object, falling back to default timezone", "user_id", userID, "error", err, "default_timezone", constants.DefaultTimezone)
		return constants.DefaultTimezone
	}

	tz := constants.DefaultTimezone
	source := "default"

	automaticTimezone, aok := user.Timezone["automaticTimezone"]
	useAutomaticTimezone, uok := user.Timezone["useAutomaticTimezone"]
	manualTimezone, mok := user.Timezone["manualTimezone"]

	if aok && uok && automaticTimezone != "" && useAutomaticTimezone == "true" {
		tz = automaticTimezone
		source = "automatic"
	} else if mok && manualTimezone != "" {
		tz = manualTimezone
		source = "manual"
	}

	s.logger.Debug("Determined user timezone", "user_id", userID, "timezone", tz, "source", source)
	return tz
}

func (s *ScheduleService) validateRequest(userID, text string) *model.CommandResponse {
	s.logger.Debug("Starting request validation", "user_id", userID)
	if maxUserMessagesErr := s.checkMaxUserMessages(userID); maxUserMessagesErr != nil {
		return s.errorResponse(formatter.FormatScheduleValidationError(maxUserMessagesErr))
	}
	if maxMessageBytesErr := s.checkMaxMessageBytes(text); maxMessageBytesErr != nil {
		return s.errorResponse(formatter.FormatScheduleValidationError(maxMessageBytesErr))
	}
	trimmedText := strings.TrimSpace(text)
	if trimmedText == "" {
		s.logger.Debug("Validation failed: empty command text", "user_id", userID)
		return s.errorResponse(formatter.FormatEmptyCommandError())
	}
	s.logger.Debug("Request validation successful", "user_id", userID)
	return nil
}

func (s *ScheduleService) persist(userID string, msg *types.ScheduledMessage) error {
	s.logger.Debug("Attempting to save scheduled message to store", "user_id", userID, "message_id", msg.ID)
	err := s.store.SaveScheduledMessage(userID, msg)
	if err == nil {
		s.logger.Debug("Successfully saved scheduled message", "user_id", userID, "message_id", msg.ID)
	}
	return err
}

func (s *ScheduleService) errorResponse(text string) *model.CommandResponse {
	s.logger.Debug("Formatting error response for user", "response_text", text)
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}
}

func (s *ScheduleService) prepareSchedule(userID, channelID, text string) (*types.ScheduledMessage, *time.Location, string, error) {
	s.logger.Debug("Preparing schedule", "user_id", userID, "channel_id", channelID)

	s.logger.Debug("Parsing schedule input text", "user_id", userID, "text", text)
	parsed, parseErr := parseScheduleInput(text)
	if parseErr != nil {
		s.logger.Error("Failed to parse schedule input", "user_id", userID, "text", text, "error", parseErr)
		return nil, nil, "", fmt.Errorf("failed to parse input: %w", parseErr)
	}
	s.logger.Debug("Parsed schedule input", "user_id", userID, "parsed_time", parsed.TimeStr, "parsed_date", parsed.DateStr, "message", parsed.Message)

	tz := s.getUserTimezone(userID)
	s.logger.Debug("Loading location based on timezone", "user_id", userID, "timezone", tz)
	loc, locErr := time.LoadLocation(tz)
	if locErr != nil {
		s.logger.Warn("Failed to load timezone location, proceeding with UTC", "user_id", userID, "timezone", tz, "error", locErr)
		loc, _ = time.LoadLocation(constants.DefaultTimezone)
	}

	now := s.clock.Now().In(loc)
	s.logger.Debug("Resolving scheduled time", "user_id", userID, "parsed_time", parsed.TimeStr, "parsed_date", parsed.DateStr, "current_time_in_loc", now, "location", loc.String())
	schedTime, resolveErr := resolveScheduledTime(parsed.TimeStr, parsed.DateStr, now, loc)
	if resolveErr != nil {
		s.logger.Error("Failed to resolve scheduled time", "user_id", userID, "parsed_time", parsed.TimeStr, "parsed_date", parsed.DateStr, "error", resolveErr)
		return nil, nil, "", fmt.Errorf("failed to resolve time: %w", resolveErr)
	}
	s.logger.Debug("Resolved scheduled time", "user_id", userID, "scheduled_time_local", schedTime, "scheduled_time_utc", schedTime.UTC())

	msgID := s.store.GenerateMessageID()
	msg := &types.ScheduledMessage{
		ID:             msgID,
		UserID:         userID,
		ChannelID:      channelID,
		PostAt:         schedTime.UTC(),
		MessageContent: parsed.Message,
		Timezone:       tz,
	}
	s.logger.Debug("Prepared scheduled message object", "user_id", userID, "message_id", msg.ID, "channel_id", msg.ChannelID, "post_at_utc", msg.PostAt, "timezone", msg.Timezone)
	return msg, loc, tz, nil
}

func (s *ScheduleService) successResponse(msg *types.ScheduledMessage, localTime time.Time, tz, channelID string) *model.CommandResponse {
	s.logger.Debug("Formatting success response", "user_id", msg.UserID, "message_id", msg.ID, "channel_id", channelID, "timezone", tz)
	channelLink := s.channel.MakeChannelLink(s.channel.GetInfoOrUnknown(channelID))
	text := formatter.FormatScheduleSuccess(localTime, tz, channelLink)
	s.logger.Debug("Formatted success response text", "user_id", msg.UserID, "message_id", msg.ID, "response_text", text)
	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}
}
