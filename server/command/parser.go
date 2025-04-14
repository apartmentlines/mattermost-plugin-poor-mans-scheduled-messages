package command

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type ParsedSchedule struct {
	TimeStr string
	DateStr string
	Message string
}

func parseScheduleInput(input string) (*ParsedSchedule, error) {
	input = strings.TrimSpace(input)
	if !strings.Contains(input, "at ") || !strings.Contains(input, " message ") {
		return nil, errors.New("invalid format. Use: `at <time> [on <date>] message <text>`")
	}

	parts := strings.SplitN(input, " message ", 2)
	if len(parts) != 2 {
		return nil, errors.New("missing message content")
	}
	before, message := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])

	if !strings.HasPrefix(before, "at ") {
		return nil, errors.New("missing 'at <time>' clause")
	}

	timePart := strings.TrimPrefix(before, "at ")
	dateStr := ""

	if strings.Contains(timePart, " on ") {
		timeDateParts := strings.SplitN(timePart, " on ", 2)
		timePart = strings.TrimSpace(timeDateParts[0])
		dateStr = strings.TrimSpace(timeDateParts[1])
	}

	if timePart == "" || message == "" {
		return nil, errors.New("time or message content missing")
	}

	return &ParsedSchedule{
		TimeStr: timePart,
		DateStr: dateStr,
		Message: message,
	}, nil
}

func resolveScheduledTime(timeStr, dateStr string, now time.Time, loc *time.Location) (time.Time, error) {
	var layouts = []string{"15:04", "3:04PM", "3:04pm", "3pm"}
	var parsedTime time.Time
	var err error
	for _, layout := range layouts {
		parsedTime, err = time.ParseInLocation(layout, timeStr, loc)
		if err == nil {
			break
		}
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("could not parse time: %v", err)
	}

	year, month, day := now.Date()
	if dateStr != "" {
		dt, err := time.ParseInLocation("2006-01-02", dateStr, loc)
		if err != nil {
			return time.Time{}, fmt.Errorf("could not parse date: %v", err)
		}
		year, month, day = dt.Date()
	} else {
		candidate := time.Date(year, month, day, parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)
		if !candidate.After(now) {
			tomorrow := now.Add(24 * time.Hour)
			year, month, day = tomorrow.Date()
		}
	}

	scheduled := time.Date(year, month, day, parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)
	if scheduled.Before(now) {
		return time.Time{}, errors.New("scheduled time is in the past")
	}
	return scheduled, nil
}
