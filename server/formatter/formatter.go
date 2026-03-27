// Package formatter contains user-facing message formatting helpers.
package formatter

import (
	"fmt"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/i18n"
)

// FormatScheduleSuccess renders a success message for scheduling.
func FormatScheduleSuccess(postAt time.Time, tz, channelLink, locale string, military bool) string {
	ts := FormatUserFacingDateTime(postAt, locale, military)
	return i18n.ScheduleSuccess(locale, constants.EmojiSuccess, ts, tz, channelLink)
}

// FormatEmptyCommandError renders a message for empty input.
func FormatEmptyCommandError() string {
	helpCommand := fmt.Sprintf("/%s %s", constants.CommandTrigger, constants.SubcommandHelp)
	return fmt.Sprintf(constants.EmptyScheduleMessage, helpCommand)
}

// FormatScheduleValidationError renders a validation error message.
func FormatScheduleValidationError(err error) string {
	return fmt.Sprintf("%s Error scheduling message: %v", constants.EmojiError, err)
}

// FormatScheduleError renders a scheduling error message.
func FormatScheduleError(postAt time.Time, tz, channelLink string, err error, locale string, military bool) string {
	ts := FormatUserFacingDateTime(postAt, locale, military)
	return i18n.SchedulePersistError(locale, constants.EmojiError, ts, tz, channelLink, err)
}

// FormatSchedulerFailure renders a scheduler failure DM message.
func FormatSchedulerFailure(channelLink string, postErr error, originalMsg string, locale string) string {
	return i18n.SchedulerFailure(locale, constants.EmojiError, channelLink, postErr, originalMsg)
}

// FormatListAttachmentHeader renders list attachment header text.
func FormatListAttachmentHeader(postAt time.Time, channelLink, messageContent, locale string, military bool) string {
	ts := FormatUserFacingDateTime(postAt, locale, military)
	return fmt.Sprintf("##### %s\n%s\n\n%s", ts, channelLink, messageContent)
}
