package formatter

import (
	"fmt"
	"time"
)

const TimeLayout = "Jan 2, 2006 3:04 PM"

func FormatScheduleSuccess(postAt time.Time, tz, channelLink string) string {
	return fmt.Sprintf("✅ Scheduled message for %s (%s) %s", postAt.Format(TimeLayout), tz, channelLink)
}

func FormatScheduleError(postAt time.Time, tz, channelLink string, err error) string {
	return fmt.Sprintf("❌ Error scheduling message for %s (%s) %s:  %v", postAt.Format(TimeLayout), tz, channelLink, err)
}

func FormatSchedulerFailure(channelLink string, postErr error, originalMsg string) string {
	return fmt.Sprintf("❌ Error scheduling message %s: %v -- original message: %s", channelLink, postErr, originalMsg)
}

func FormatListAttachmentHeader(postAt time.Time, channelLink, messageContent string) string {
	return fmt.Sprintf("##### %s\n%s\n\n%s", postAt.Format(TimeLayout), channelLink, messageContent)
}
