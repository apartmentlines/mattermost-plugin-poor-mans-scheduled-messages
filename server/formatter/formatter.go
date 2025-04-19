package formatter

import (
	"fmt"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
)

func FormatScheduleSuccess(postAt time.Time, tz, channelLink string) string {
	return fmt.Sprintf("%s Scheduled message for %s (%s) %s", constants.EmojiSuccess, postAt.Format(constants.TimeLayout), tz, channelLink)
}

func FormatEmptyCommandError() string {
	helpCommand := fmt.Sprintf("/%s %s", constants.CommandTrigger, constants.SubcommandHelp)
	return fmt.Sprintf(constants.EmptyScheduleMessage, helpCommand)
}

func FormatScheduleValidationError(err error) string {
	return fmt.Sprintf("%s Error scheduling message: %v", constants.EmojiError, err)
}

func FormatScheduleError(postAt time.Time, tz, channelLink string, err error) string {
	return fmt.Sprintf("%s Error scheduling message for %s (%s) %s:  %v", constants.EmojiError, postAt.Format(constants.TimeLayout), tz, channelLink, err)
}

func FormatSchedulerFailure(channelLink string, postErr error, originalMsg string) string {
	return fmt.Sprintf("%s Error scheduling message %s: %v -- original message: %s", constants.EmojiError, channelLink, postErr, originalMsg)
}

func FormatListAttachmentHeader(postAt time.Time, channelLink, messageContent string) string {
	return fmt.Sprintf("##### %s\n%s\n\n%s", postAt.Format(constants.TimeLayout), channelLink, messageContent)
}
