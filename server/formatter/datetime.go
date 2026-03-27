package formatter

import (
	"fmt"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/i18n"
)

var ukMonthsGenitive = map[time.Month]string{
	time.January:   "січня",
	time.February:  "лютого",
	time.March:     "березня",
	time.April:     "квітня",
	time.May:       "травня",
	time.June:      "червня",
	time.July:      "липня",
	time.August:    "серпня",
	time.September: "вересня",
	time.October:   "жовтня",
	time.November:  "листопада",
	time.December:  "грудня",
}

var ruMonthsGenitive = map[time.Month]string{
	time.January:   "января",
	time.February:  "февраля",
	time.March:     "марта",
	time.April:     "апреля",
	time.May:       "мая",
	time.June:      "июня",
	time.July:      "июля",
	time.August:    "августа",
	time.September: "сентября",
	time.October:   "октября",
	time.November:  "ноября",
	time.December:  "декабря",
}

// FormatUserFacingDateTime formats date and time using the user's locale and 12h/24h preference.
func FormatUserFacingDateTime(t time.Time, locale string, military bool) string {
	switch i18n.NormalizeLocale(locale) {
	case "uk":
		return formatCyrillicDateTime(t, ukMonthsGenitive, military)
	case "ru":
		return formatCyrillicDateTime(t, ruMonthsGenitive, military)
	default:
		if military {
			return t.Format("Jan 2, 2006 15:04")
		}
		return t.Format("Jan 2, 2006 3:04 PM")
	}
}

func formatCyrillicDateTime(t time.Time, months map[time.Month]string, military bool) string {
	mon := months[t.Month()]
	datePart := fmt.Sprintf("%d %s %d", t.Day(), mon, t.Year())
	if military {
		return fmt.Sprintf("%s %02d:%02d", datePart, t.Hour(), t.Minute())
	}
	h := t.Hour()
	h12 := h % 12
	if h12 == 0 {
		h12 = 12
	}
	suffix := "дп"
	if h >= 12 {
		suffix = "пп"
	}
	return fmt.Sprintf("%s %d:%02d %s", datePart, h12, t.Minute(), suffix)
}
