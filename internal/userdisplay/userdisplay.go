// Package userdisplay resolves Mattermost user locale and clock preferences.
package userdisplay

import (
	"strings"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

// FromAPI reads the user's locale and 24-hour (military) time preference.
// api may be nil (e.g. in lightweight unit tests); defaults are English, 12-hour clock.
func FromAPI(api plugin.API, userID string) (locale string, military bool) {
	locale = "en"
	military = false
	if api == nil {
		return locale, military
	}

	if user, uerr := api.GetUser(userID); uerr == nil && user != nil && strings.TrimSpace(user.Locale) != "" {
		locale = user.Locale
	}

	prefs, perr := api.GetPreferencesForUser(userID)
	if perr != nil {
		return locale, military
	}
	for _, pr := range prefs {
		if pr.Category == model.PreferenceCategoryDisplaySettings && pr.Name == model.PreferenceNameUseMilitaryTime && pr.Value == "true" {
			military = true
			break
		}
	}
	return locale, military
}
