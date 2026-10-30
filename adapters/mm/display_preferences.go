package mm

import (
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
)

// DisplayPreferences reads live settings without caching them on scheduled jobs.
type DisplayPreferences struct{ api ports.DisplaySettingsAPI }

// NewDisplayPreferences constructs the Mattermost display settings adapter.
func NewDisplayPreferences(api ports.DisplaySettingsAPI) *DisplayPreferences {
	return &DisplayPreferences{api: api}
}

// ForUser falls back independently for unavailable locale and clock settings.
func (p *DisplayPreferences) ForUser(userID string) types.DisplaySettings {
	settings := types.DisplaySettings{Locale: "en"}
	if user, err := p.api.GetUser(userID); err == nil && user != nil && user.Locale != "" {
		settings.Locale = user.Locale
	} else if config := p.api.GetConfig(); config != nil && config.LocalizationSettings.DefaultClientLocale != nil {
		settings.Locale = *config.LocalizationSettings.DefaultClientLocale
	}
	if prefs, err := p.api.GetPreferencesForUser(userID); err == nil {
		for _, pref := range prefs {
			if pref.Category == model.PreferenceCategoryDisplaySettings && pref.Name == model.PreferenceNameUseMilitaryTime {
				settings.Use24Hour = pref.Value == "true"
				break
			}
		}
	}
	return settings
}
