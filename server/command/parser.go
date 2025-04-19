package command

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
)

type dateFormat int

const (
	dateFormatInvalid dateFormat = iota
	dateFormatNone
	dateFormatYYYYMMDD
	dateFormatDayOfWeek
	dateFormatShortDayMonth
)

var (
	regexFullCommand    = regexp.MustCompile(`(?i)^at\s+([0-9]{1,2}(?::[0-9]{2})?\s*(?:am|pm)?)(?:\s+on\s+((?:\d{4}-\d{2}-\d{2})|(?:\d{1,2}[a-z]{3})|(?:mon|tue|wed|thu|fri|sat|sun)|(?:monday|tuesday|wednesday|thursday|friday|saturday|sunday)))?\s+message\s+(.+)$`)
	regexpYYYYMMDD      = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	regexpShortDayMonth = regexp.MustCompile(`^(\d{1,2})([a-z]{3})$`)
)

// Maps for parsing day and month names/abbreviations.
var (
	dayOfWeekMap = map[string]time.Weekday{
		"sunday":    time.Sunday,
		"sun":       time.Sunday,
		"monday":    time.Monday,
		"mon":       time.Monday,
		"tuesday":   time.Tuesday,
		"tue":       time.Tuesday,
		"wednesday": time.Wednesday,
		"wed":       time.Wednesday,
		"thursday":  time.Thursday,
		"thu":       time.Thursday,
		"friday":    time.Friday,
		"fri":       time.Friday,
		"saturday":  time.Saturday,
		"sat":       time.Saturday,
	}
	monthAbbrMap = map[string]time.Month{
		"jan": time.January,
		"feb": time.February,
		"mar": time.March,
		"apr": time.April,
		"may": time.May,
		"jun": time.June,
		"jul": time.July,
		"aug": time.August,
		"sep": time.September,
		"oct": time.October,
		"nov": time.November,
		"dec": time.December,
	}
)

type ParsedSchedule struct {
	TimeStr string
	DateStr string
	Message string
}

func parseScheduleInput(input string) (*ParsedSchedule, error) {
	trimmedInput := strings.TrimSpace(input)
	matches := regexFullCommand.FindStringSubmatch(trimmedInput)
	if matches == nil {
		return nil, errors.New(constants.ParserErrInvalidFormat)
	}
	timeStr := strings.ToLower(strings.ReplaceAll(matches[1], " ", ""))
	if len(timeStr) > 1 && timeStr[0] == '0' {
		timeStr = timeStr[1:]
	}
	dateStr := strings.ToLower(matches[2])
	message := matches[3]
	return &ParsedSchedule{
		TimeStr: timeStr,
		DateStr: dateStr,
		Message: message,
	}, nil
}

func determineDateFormat(dateStr string) dateFormat {
	if dateStr == "" {
		return dateFormatNone
	}
	if regexpYYYYMMDD.MatchString(dateStr) {
		return dateFormatYYYYMMDD
	}
	if _, dayOfWeekOk := dayOfWeekMap[dateStr]; dayOfWeekOk {
		return dateFormatDayOfWeek
	}
	if matches := regexpShortDayMonth.FindStringSubmatch(dateStr); matches != nil {
		if _, monthOk := monthAbbrMap[matches[2]]; monthOk {
			dayInt, dayErr := strconv.Atoi(matches[1])
			if dayErr == nil && dayInt >= 1 && dayInt <= 31 {
				return dateFormatShortDayMonth
			}
		}
	}
	return dateFormatInvalid
}

func resolveDateTimeNone(parsedTime time.Time, now time.Time, loc *time.Location) (time.Time, error) {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	candidateDateTimeToday := time.Date(today.Year(), today.Month(), today.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)
	if candidateDateTimeToday.After(now) {
		return candidateDateTimeToday, nil
	}
	tomorrow := today.AddDate(0, 0, 1)
	candidateDateTimeTomorrow := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)
	return candidateDateTimeTomorrow, nil
}

func resolveDateTimeYYYYMMDD(dateStr string, parsedTime time.Time, now time.Time, loc *time.Location) (time.Time, error) {
	parsedDatePart, err := time.ParseInLocation(constants.DateParseLayoutYYYYMMDD, dateStr, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date specified '%s': %w", dateStr, err)
	}
	scheduledTime := time.Date(parsedDatePart.Year(), parsedDatePart.Month(), parsedDatePart.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)
	if !scheduledTime.After(now) {
		return time.Time{}, fmt.Errorf("scheduled time '%s' for date '%s' is already in the past -- must be in the future when using a specific date", parsedTime.Format("3:15pm"), dateStr)
	}
	return scheduledTime, nil
}

func resolveDateTimeDayOfWeek(dateStr string, parsedTime time.Time, now time.Time, loc *time.Location) (time.Time, error) {
	targetWeekday, ok := dayOfWeekMap[dateStr]
	if !ok {
		return time.Time{}, fmt.Errorf("invalid day of week '%s'", dateStr)
	}
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	currentWeekday := now.Weekday()
	daysToAdd := int((targetWeekday - currentWeekday + 7) % 7)
	candidateDate := today.AddDate(0, 0, daysToAdd)
	candidateDateTime := time.Date(candidateDate.Year(), candidateDate.Month(), candidateDate.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)
	if !candidateDateTime.After(now) {
		nextWeekDate := candidateDate.AddDate(0, 0, 7)
		candidateDateTime = time.Date(nextWeekDate.Year(), nextWeekDate.Month(), nextWeekDate.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)
	}
	return candidateDateTime, nil
}

func resolveDateTimeShortDayMonth(dateStr string, parsedTime time.Time, now time.Time, loc *time.Location) (time.Time, error) {
	matches := regexpShortDayMonth.FindStringSubmatch(dateStr)
	if matches == nil {
		return time.Time{}, fmt.Errorf("invalid short day/month format '%s'", dateStr)
	}
	dayInt, _ := strconv.Atoi(matches[1])
	monthAbbr := matches[2]
	targetMonth, monthOk := monthAbbrMap[monthAbbr]
	if !monthOk {
		return time.Time{}, fmt.Errorf("invalid month '%s'", monthAbbr)
	}
	createAndValidateDate := func(year int) (time.Time, error) {
		d := time.Date(year, targetMonth, dayInt, 0, 0, 0, 0, loc)
		if d.Day() != dayInt || d.Month() != targetMonth || d.Year() != year {
			return time.Time{}, fmt.Errorf("invalid date specified: %d%s", dayInt, monthAbbr)
		}
		return d, nil
	}
	candidateDateThisYear, err := createAndValidateDate(now.Year())
	if err != nil {
		return time.Time{}, err
	}
	candidateDateTimeThisYear := time.Date(candidateDateThisYear.Year(), candidateDateThisYear.Month(), candidateDateThisYear.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)
	if candidateDateTimeThisYear.After(now) {
		return candidateDateTimeThisYear, nil
	}
	candidateDateNextYear, err := createAndValidateDate(now.Year() + 1)
	if err != nil {
		return time.Time{}, err
	}
	candidateDateTimeNextYear := time.Date(candidateDateNextYear.Year(), candidateDateNextYear.Month(), candidateDateNextYear.Day(), parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)
	return candidateDateTimeNextYear, nil
}

func parseTimeStr(timeStr string, loc *time.Location) (time.Time, error) {
	for _, layout := range constants.TimeParseLayouts {
		parsedTime, err := time.ParseInLocation(layout, timeStr, loc)
		if err == nil {
			return parsedTime, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse time: '%s'. Use formats like 9:30AM, 17:00, 5pm", timeStr)
}

func resolveScheduledTime(timeStr string, dateStr string, now time.Time, loc *time.Location) (time.Time, error) {
	parsedTime, parseTimeErr := parseTimeStr(timeStr, loc)
	if parseTimeErr != nil {
		return parsedTime, parseTimeErr
	}
	format := determineDateFormat(dateStr)
	switch format {
	case dateFormatNone:
		return resolveDateTimeNone(parsedTime, now, loc)
	case dateFormatYYYYMMDD:
		return resolveDateTimeYYYYMMDD(dateStr, parsedTime, now, loc)
	case dateFormatDayOfWeek:
		return resolveDateTimeDayOfWeek(dateStr, parsedTime, now, loc)
	case dateFormatShortDayMonth:
		return resolveDateTimeShortDayMonth(dateStr, parsedTime, now, loc)
	case dateFormatInvalid:
		return time.Time{}, fmt.Errorf(constants.ParserErrInvalidDateFormat, dateStr)
	default:
		return time.Time{}, errors.New(constants.ParserErrUnknownDateFormat)
	}
}
