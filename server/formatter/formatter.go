// Package formatter contains user-facing message formatting helpers.
package formatter

import (
	"fmt"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
)

// FormatScheduleSuccess renders a success message for scheduling.
func (r Renderer) FormatScheduleSuccess(postAt time.Time, tz, channelLink string, inThread bool) string {
	return r.Text("scheduled", constants.EmojiSuccess, r.DateTime(postAt), tz, r.formatDestination(channelLink, inThread))
}

// FormatEmptyCommandError renders a message for empty input.
func (r Renderer) FormatEmptyCommandError() string {
	helpCommand := fmt.Sprintf("/%s %s", constants.CommandTrigger, constants.SubcommandHelp)
	return r.Text("empty_command", helpCommand)
}

// FormatScheduleValidationError renders a validation error message.
func (r Renderer) FormatScheduleValidationError(err error) string {
	return r.Text("validation", constants.EmojiError, err)
}

// FormatScheduleError renders a scheduling error message.
func (r Renderer) FormatScheduleError(postAt time.Time, tz, channelLink string, err error) string {
	return r.Text("schedule_error", constants.EmojiError, r.DateTime(postAt), tz, channelLink, err)
}

// FormatSchedulerFailure renders a scheduler failure DM message.
func (r Renderer) FormatSchedulerFailure(channelLink string, postErr error, originalMsg string) string {
	return r.Text("failure", constants.EmojiError, channelLink, postErr, originalMsg)
}

// FormatListAttachmentHeader renders list attachment header text.
func (r Renderer) FormatListAttachmentHeader(postAt time.Time, channelLink, messageContent string, inThread bool) string {
	return fmt.Sprintf("##### %s\n%s\n\n%s", r.DateTime(postAt), r.formatDestination(channelLink, inThread), messageContent)
}

func (r Renderer) formatDestination(channelLink string, inThread bool) string {
	if inThread {
		return channelLink + " " + r.Text("thread")
	}
	return channelLink
}
