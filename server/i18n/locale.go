package i18n

import "strings"

// NormalizeLocale maps Mattermost locales to bundled languages. Unknown values fall back to English.
func NormalizeLocale(locale string) string {
	l := strings.TrimSpace(strings.ToLower(locale))
	switch {
	case strings.HasPrefix(l, "uk"):
		return "uk"
	case strings.HasPrefix(l, "ru"):
		return "ru"
	default:
		return "en"
	}
}
