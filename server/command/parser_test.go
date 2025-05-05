package command

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
)

func TestParseScheduleInput(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        *ParsedSchedule
		wantErr     bool
		errContains string
	}{
		{
			name:  "Basic no-date",
			input: "at 3pm message Hello world",
			want:  &ParsedSchedule{TimeStr: "3pm", DateStr: "", Message: "Hello world"},
		},
		{
			name:  "With 24h time no-date",
			input: "AT 15:04 message Test",
			want:  &ParsedSchedule{TimeStr: "15:04", DateStr: "", Message: "Test"},
		},
		{
			name:  "With explicit YYYY-MM-DD",
			input: "at 03:05am on 2025-12-31 message Year end",
			want:  &ParsedSchedule{TimeStr: "3:05am", DateStr: "2025-12-31", Message: "Year end"},
		},
		{
			name:  "With short day/month",
			input: "at 9:30PM on 5feb message Short",
			want:  &ParsedSchedule{TimeStr: "9:30pm", DateStr: "5feb", Message: "Short"},
		},
		{
			name:  "With full day-name",
			input: "at 10pm on Tuesday message Weekday",
			want:  &ParsedSchedule{TimeStr: "10pm", DateStr: "tuesday", Message: "Weekday"},
		},
		{
			name:  "Extra spaces and mixed case",
			input: "  aT   5 pm   On   MonDay    mEsSaGe   Mixed Case ",
			want:  &ParsedSchedule{TimeStr: "5pm", DateStr: "monday", Message: "Mixed Case"},
		},
		{
			name:  "Multi-line message",
			input: "at 10am message Line 1\nLine 2",
			want:  &ParsedSchedule{TimeStr: "10am", DateStr: "", Message: "Line 1\nLine 2"},
		},
		{
			name:  "Newline separator after message keyword",
			input: "at 11am message\nStarts on new line",
			want:  &ParsedSchedule{TimeStr: "11am", DateStr: "", Message: "Starts on new line"},
		},
		{
			name:  "Mixed whitespace separator after message keyword",
			input: "at 1pm message \t \n Starts indented",
			want:  &ParsedSchedule{TimeStr: "1pm", DateStr: "", Message: "Starts indented"},
		},
		{
			name:  "Message with leading/trailing newlines",
			input: "at 2pm message \n\n Content \n\n",
			want:  &ParsedSchedule{TimeStr: "2pm", DateStr: "", Message: "Content"},
		},
		{
			name:        "Message with only whitespace/newlines",
			input:       "at 4pm message \n \t \n ",
			wantErr:     true,
			errContains: constants.ParserErrInvalidFormat,
		},
		{
			name:  "Date included with multi-line message",
			input: "at 6pm on 2024-08-15 message First line\nSecond line",
			want:  &ParsedSchedule{TimeStr: "6pm", DateStr: "2024-08-15", Message: "First line\nSecond line"},
		},
		{
			name:        "Missing 'message' keyword",
			input:       "at 3pm on mon foo bar",
			wantErr:     true,
			errContains: constants.ParserErrInvalidFormat,
		},
		{
			name:        "Missing 'at' keyword",
			input:       "3pm message hello",
			wantErr:     true,
			errContains: constants.ParserErrInvalidFormat,
		},
		{
			name:        "Empty input",
			input:       "",
			wantErr:     true,
			errContains: constants.ParserErrInvalidFormat,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ps, err := parseScheduleInput(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tc.input)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("error %v does not contain %q", err, tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}
			if ps.TimeStr != tc.want.TimeStr {
				t.Errorf("TimeStr = %q, want %q", ps.TimeStr, tc.want.TimeStr)
			}
			if ps.DateStr != tc.want.DateStr {
				t.Errorf("DateStr = %q, want %q", ps.DateStr, tc.want.DateStr)
			}
			if ps.Message != tc.want.Message {
				t.Errorf("Message = %q, want %q", ps.Message, tc.want.Message)
			}
		})
	}
}

func TestDetermineDateFormat(t *testing.T) {
	tests := []struct {
		input string
		want  dateFormat
	}{
		{"", dateFormatNone},
		{"2022-01-02", dateFormatYYYYMMDD},
		{"mon", dateFormatDayOfWeek},
		{"MONDAY", dateFormatDayOfWeek},
		{"wed", dateFormatDayOfWeek},
		{"Friday", dateFormatDayOfWeek},
		{"3jan", dateFormatShortDayMonth},
		{"25Dec", dateFormatShortDayMonth},
		{"32jan", dateFormatInvalid},
		{"15xyz", dateFormatInvalid},
		{"feb31", dateFormatInvalid},
		{"foo", dateFormatInvalid},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := determineDateFormat(strings.ToLower(tc.input))
			if got != tc.want {
				t.Errorf("determineDateFormat(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseTimeStr(t *testing.T) {
	loc := time.UTC
	tests := []struct {
		input       string
		wantHour    int
		wantMinute  int
		wantErr     bool
		errContains string
	}{
		{"15:04", 15, 4, false, ""},
		{"3:04pm", 15, 4, false, ""},
		{"3:04PM", 15, 4, false, ""},
		{"3pm", 15, 0, false, ""},
		{"3PM", 15, 0, false, ""},
		{"12am", 0, 0, false, ""},
		{"12:00", 12, 0, false, ""},
		{"5pm", 17, 0, false, ""},
		{"24:00", 0, 0, true, "could not parse time"},
		{"3:60pm", 0, 0, true, "could not parse time"},
		{"abc", 0, 0, true, "could not parse time"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			pt, err := parseTimeStr(tc.input, loc)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for input %q", tc.input)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("error %v does not contain %q", err, tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for input %q: %v", tc.input, err)
			}
			if pt.Hour() != tc.wantHour || pt.Minute() != tc.wantMinute {
				t.Errorf("parseTimeStr(%q) = %02d:%02d, want %02d:%02d", tc.input, pt.Hour(), pt.Minute(), tc.wantHour, tc.wantMinute)
			}
		})
	}
}

func TestResolveDateTimeNone(t *testing.T) {
	loc := time.UTC
	now := time.Date(2024, time.January, 1, 14, 0, 0, 0, loc)
	tests := []struct {
		parsedTime time.Time
		now        time.Time
		want       time.Time
	}{
		{time.Date(2024, time.January, 1, 15, 0, 0, 0, loc), now, time.Date(2024, time.January, 1, 15, 0, 0, 0, loc)},
		{time.Date(2024, time.January, 1, 14, 0, 0, 0, loc), now, time.Date(2024, time.January, 2, 14, 0, 0, 0, loc)},
		{time.Date(2024, time.January, 1, 0, 0, 0, 0, loc), now, time.Date(2024, time.January, 2, 0, 0, 0, 0, loc)},
	}

	for _, tc := range tests {
		t.Run(tc.want.String(), func(t *testing.T) {
			got, err := resolveDateTimeNone(tc.parsedTime, tc.now, loc)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveDateTimeYYYYMMDD(t *testing.T) {
	loc := time.UTC
	now := time.Date(2024, time.January, 1, 14, 0, 0, 0, loc)
	tests := []struct {
		dateStr     string
		parsedTime  time.Time
		now         time.Time
		want        time.Time
		wantErr     bool
		errContains string
	}{
		{"2024-01-02", time.Date(2024, time.January, 1, 15, 0, 0, 0, loc), now, time.Date(2024, time.January, 2, 15, 0, 0, 0, loc), false, ""},
		{"2024-01-01", time.Date(2024, time.January, 1, 15, 0, 0, 0, loc), now, time.Date(2024, time.January, 1, 15, 0, 0, 0, loc), false, ""},
		{"2024-01-01", time.Date(2024, time.January, 1, 13, 0, 0, 0, loc), now, time.Time{}, true, "already in the past"},
		{"invalid", time.Date(2024, time.January, 1, 12, 0, 0, 0, loc), now, time.Time{}, true, "invalid date specified"},
	}

	for _, tc := range tests {
		t.Run(tc.dateStr, func(t *testing.T) {
			got, err := resolveDateTimeYYYYMMDD(tc.dateStr, tc.parsedTime, tc.now, loc)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.dateStr)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("error %v does not contain %q", err, tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.dateStr, err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveDateTimeDayOfWeek(t *testing.T) {
	loc := time.UTC
	now := time.Date(2024, time.January, 3, 14, 0, 0, 0, loc)
	tests := []struct {
		dateStr     string
		parsedTime  time.Time
		want        time.Time
		wantErr     bool
		errContains string
	}{
		{"wed", time.Date(2024, time.January, 1, 15, 0, 0, 0, loc), time.Date(2024, time.January, 3, 15, 0, 0, 0, loc), false, ""},
		{"wed", time.Date(2024, time.January, 1, 13, 0, 0, 0, loc), time.Date(2024, time.January, 10, 13, 0, 0, 0, loc), false, ""},
		{"fri", time.Date(2024, time.January, 1, 10, 0, 0, 0, loc), time.Date(2024, time.January, 5, 10, 0, 0, 0, loc), false, ""},
		{"sun", time.Date(2024, time.January, 1, 9, 0, 0, 0, loc), time.Date(2024, time.January, 7, 9, 0, 0, 0, loc), false, ""},
		{"invalid", time.Date(2024, time.January, 1, 8, 0, 0, 0, loc), time.Time{}, true, "invalid day of week"},
	}

	for _, tc := range tests {
		t.Run(tc.dateStr, func(t *testing.T) {
			got, err := resolveDateTimeDayOfWeek(tc.dateStr, tc.parsedTime, now, loc)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.dateStr)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("error %v does not contain %q", err, tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.dateStr, err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveDateTimeShortDayMonth(t *testing.T) {
	loc := time.UTC
	now := time.Date(2024, time.January, 15, 12, 0, 0, 0, loc)
	defaultParsedTime := time.Date(2024, time.January, 15, 9, 0, 0, 0, loc)
	tests := []struct {
		dateStr     string
		parsedTime  time.Time
		want        time.Time
		wantErr     bool
		errContains string
	}{
		{"9jan", defaultParsedTime, time.Date(2025, time.January, 9, 9, 0, 0, 0, loc), false, ""},
		{"15jan", time.Date(2024, time.January, 15, 18, 0, 0, 0, loc), time.Date(2024, time.January, 15, 18, 0, 0, 0, loc), false, ""},
		{"15jan", defaultParsedTime, time.Date(2025, time.January, 15, 9, 0, 0, 0, loc), false, ""},
		{"16jan", defaultParsedTime, time.Date(2024, time.January, 16, 9, 0, 0, 0, loc), false, ""},
		{"20jan", defaultParsedTime, time.Date(2024, time.January, 20, 9, 0, 0, 0, loc), false, ""},
		// Leap year.
		{"29feb", defaultParsedTime, time.Date(2024, time.February, 29, 9, 0, 0, 0, loc), false, ""},
		{"xxx", defaultParsedTime, time.Time{}, true, "invalid short day/month format 'xxx'"},
		{"15foo", defaultParsedTime, time.Time{}, true, "invalid month 'foo'"},
		{"31apr", defaultParsedTime, time.Time{}, true, "invalid date specified"},
	}

	for _, tc := range tests {
		t.Run(tc.dateStr, func(t *testing.T) {
			got, err := resolveDateTimeShortDayMonth(tc.dateStr, tc.parsedTime, now, loc)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %q", tc.dateStr)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("error %v does not contain %q", err, tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %q: %v", tc.dateStr, err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveScheduledTime(t *testing.T) {
	loc := time.UTC
	now := time.Date(2024, time.January, 1, 14, 0, 0, 0, loc)
	tests := []struct {
		name        string
		timeStr     string
		dateStr     string
		now         time.Time
		want        time.Time
		wantErr     bool
		errContains string
	}{
		{"no date", "3pm", "", now, time.Date(2024, time.January, 1, 15, 0, 0, 0, loc), false, ""},
		{"specific date ok", "3pm", "2024-01-01", now, time.Date(2024, time.January, 1, 15, 0, 0, 0, loc), false, ""},
		{"specific date past", "3pm", "2024-01-01", time.Date(2024, time.January, 1, 16, 0, 0, 0, loc), time.Time{}, true, "already in the past"},
		{"invalid time", "invalid", "", now, time.Time{}, true, "could not parse time"},
		{"invalid date", "3pm", "foo", now, time.Time{}, true, fmt.Sprintf(constants.ParserErrInvalidDateFormat, "foo")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveScheduledTime(tc.timeStr, tc.dateStr, tc.now, loc)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error for %s", tc.name)
				}
				if !strings.Contains(err.Error(), tc.errContains) {
					t.Fatalf("error %v does not contain %q", err, tc.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.name, err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
