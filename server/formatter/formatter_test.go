package formatter

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
)

func TestFormatScheduleSuccess(t *testing.T) {
	ts := time.Date(2025, time.January, 2, 15, 4, 0, 0, time.UTC)
	tz := "UTC"
	channel := "in channel: ~town-square"

	expected := fmt.Sprintf("%s Scheduled message for %s (%s) %s", constants.EmojiSuccess, ts.Format(constants.TimeLayout), tz, channel)

	got := FormatScheduleSuccess(ts, tz, channel)
	if got != expected {
		t.Fatalf("FormatScheduleSuccess() = %q, want %q", got, expected)
	}
}

func TestFormatEmptyCommandError(t *testing.T) {
	helpCommand := fmt.Sprintf("/%s %s", constants.CommandTrigger, constants.SubcommandHelp)
	expected := fmt.Sprintf(constants.EmptyScheduleMessage, helpCommand)

	got := FormatEmptyCommandError()
	if got != expected {
		t.Fatalf("FormatEmptyCommandError() = %q, want %q", got, expected)
	}
}

func TestFormatScheduleValidationError(t *testing.T) {
	errVal := errors.New("validation failure")
	expected := fmt.Sprintf("%s Error scheduling message: %v", constants.EmojiError, errVal)

	got := FormatScheduleValidationError(errVal)
	if got != expected {
		t.Fatalf("FormatScheduleValidationError() = %q, want %q", got, expected)
	}
}

func TestFormatScheduleError(t *testing.T) {
	ts := time.Date(2025, time.January, 2, 15, 4, 0, 0, time.UTC)
	tz := "UTC"
	channel := "in channel: ~town-square"
	errVal := errors.New("store failure")

	expected := fmt.Sprintf("%s Error scheduling message for %s (%s) %s:  %v", constants.EmojiError, ts.Format(constants.TimeLayout), tz, channel, errVal)

	got := FormatScheduleError(ts, tz, channel, errVal)
	if got != expected {
		t.Fatalf("FormatScheduleError() = %q, want %q", got, expected)
	}
}

func TestFormatSchedulerFailure(t *testing.T) {
	channel := "in channel: ~town-square"
	postErr := errors.New("post failure")
	orig := "hello world"

	expected := fmt.Sprintf("%s Error scheduling message %s: %v -- original message: %s", constants.EmojiError, channel, postErr, orig)

	got := FormatSchedulerFailure(channel, postErr, orig)
	if got != expected {
		t.Fatalf("FormatSchedulerFailure() = %q, want %q", got, expected)
	}
}

func TestFormatListAttachmentHeader(t *testing.T) {
	ts := time.Date(2025, time.January, 2, 15, 4, 0, 0, time.UTC)
	channel := "in channel: ~town-square"
	msg := "hello world"

	expected := fmt.Sprintf("##### %s\n%s\n\n%s", ts.Format(constants.TimeLayout), channel, msg)

	got := FormatListAttachmentHeader(ts, channel, msg)
	if got != expected {
		t.Fatalf("FormatListAttachmentHeader() = %q, want %q", got, expected)
	}
}
