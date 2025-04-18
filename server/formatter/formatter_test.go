package formatter

import (
	"errors"
	"testing"
	"time"
)

func TestFormatScheduleSuccess(t *testing.T) {
	ts := time.Date(2025, time.January, 2, 15, 4, 0, 0, time.UTC)
	tz := "UTC"
	channel := "in channel: ~town-square"

	expected := "✅ Scheduled message for " + ts.Format(TimeLayout) + " (" + tz + ") " + channel

	got := FormatScheduleSuccess(ts, tz, channel)
	if got != expected {
		t.Fatalf("FormatScheduleSuccess() = %q, want %q", got, expected)
	}
}

func TestFormatScheduleError(t *testing.T) {
	ts := time.Date(2025, time.January, 2, 15, 4, 0, 0, time.UTC)
	tz := "UTC"
	channel := "in channel: ~town-square"
	errVal := errors.New("store failure")

	expected := "❌ Error scheduling message for " + ts.Format(TimeLayout) + " (" + tz + ") " + channel + ":  " + errVal.Error()

	got := FormatScheduleError(ts, tz, channel, errVal)
	if got != expected {
		t.Fatalf("FormatScheduleError() = %q, want %q", got, expected)
	}
}

func TestFormatSchedulerFailure(t *testing.T) {
	channel := "in channel: ~town-square"
	postErr := errors.New("post failure")
	orig := "hello world"

	expected := "❌ Error scheduling message " + channel + ": " + postErr.Error() + " -- original message: " + orig

	got := FormatSchedulerFailure(channel, postErr, orig)
	if got != expected {
		t.Fatalf("FormatSchedulerFailure() = %q, want %q", got, expected)
	}
}

func TestFormatListAttachmentHeader(t *testing.T) {
	ts := time.Date(2025, time.January, 2, 15, 4, 0, 0, time.UTC)
	channel := "in channel: ~town-square"
	msg := "hello world"

	expected := "##### " + ts.Format(TimeLayout) + "\n" + channel + "\n\n" + msg

	got := FormatListAttachmentHeader(ts, channel, msg)
	if got != expected {
		t.Fatalf("FormatListAttachmentHeader() = %q, want %q", got, expected)
	}
}
