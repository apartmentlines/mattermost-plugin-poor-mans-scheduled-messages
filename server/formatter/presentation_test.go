package formatter

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/stretchr/testify/require"
)

func TestDateTimeUsesLocaleAndPersonalClockIndependently(t *testing.T) {
	for _, tc := range []struct {
		locale   string
		military bool
		hour     int
		want     string
	}{
		{"ru", true, 13, "30 октября 2026 13:10"},
		{"ru", false, 13, "30 октября 2026 1:10 PM"},
		{"en", true, 13, "Oct 30, 2026 13:10"},
		{"en", false, 13, "Oct 30, 2026 1:10 PM"},
		{"ru", true, 0, "30 октября 2026 00:10"},
		{"en", false, 0, "Oct 30, 2026 12:10 AM"},
		{"en", false, 12, "Oct 30, 2026 12:10 PM"},
		{"en-AU", false, 13, "30 Oct 2026 1:10 PM"},
		{"ru-RU", true, 13, "30 октября 2026 13:10"},
		{"unknown", true, 13, "Oct 30, 2026 13:10"},
	} {
		t.Run(fmt.Sprintf("%s/%t/%d", tc.locale, tc.military, tc.hour), func(t *testing.T) {
			r := Renderer{types.DisplaySettings{Locale: tc.locale, Use24Hour: tc.military}}
			require.Equal(t, tc.want, r.DateTime(time.Date(2026, 10, 30, tc.hour, 10, 0, 0, time.UTC)))
		})
	}
}

func TestAllMattermostCatalogsAreComplete(t *testing.T) {
	// Keep in sync with webapp/channels/src/i18n/i18n.ts in Mattermost.
	locales := []string{"de", "en", "en-AU", "es", "fr", "it", "hu", "nl", "pl", "pt-BR", "ro", "sv", "vi", "tr", "bg", "ru", "uk", "fa", "ko", "zh-CN", "zh-TW", "ja", "da", "cs", "hr", "lt", "nb-NO", "pt", "fi"}
	verbs := regexp.MustCompile(`%[sdv]`)
	require.Len(t, catalogs, len(locales))
	for _, locale := range locales {
		t.Run(locale, func(t *testing.T) {
			c, ok := catalogs[locale]
			require.True(t, ok)
			require.Len(t, c.Months, 12)
			require.Len(t, c.Messages, len(catalogs["en"].Messages))
			for key, english := range catalogs["en"].Messages {
				require.NotEmpty(t, c.Messages[key], key)
				require.Equal(t, verbs.FindAllString(english, -1), verbs.FindAllString(c.Messages[key], -1), key)
			}
			for month := time.January; month <= time.December; month++ {
				r := Renderer{types.DisplaySettings{Locale: locale, Use24Hour: true}}
				value := r.DateTime(time.Date(2026, month, 2, 13, 10, 0, 0, time.UTC))
				require.Contains(t, value, c.Months[int(month)-1])
				require.Contains(t, value, "13:10")
			}
		})
	}
}
