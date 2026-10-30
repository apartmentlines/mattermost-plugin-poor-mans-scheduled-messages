package mm

import (
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/require"
)

func TestDisplayPreferencesRefreshAndIndependentFallbacks(t *testing.T) {
	api := &plugintest.API{}
	adapter := NewDisplayPreferences(api)
	api.On("GetUser", "u").Return(&model.User{Locale: "ru"}, nil).Once()
	api.On("GetPreferencesForUser", "u").Return([]model.Preference{
		{Category: "other", Name: model.PreferenceNameUseMilitaryTime, Value: "false"},
		{Category: model.PreferenceCategoryDisplaySettings, Name: model.PreferenceNameUseMilitaryTime, Value: "true"},
	}, nil).Once()
	settings := adapter.ForUser("u")
	require.Equal(t, "ru", settings.Locale)
	require.True(t, settings.Use24Hour)
	api.On("GetUser", "u").Return(&model.User{Locale: "de"}, nil).Once()
	api.On("GetPreferencesForUser", "u").Return([]model.Preference{{Category: model.PreferenceCategoryDisplaySettings, Name: model.PreferenceNameUseMilitaryTime, Value: "false"}}, nil).Once()
	settings = adapter.ForUser("u")
	require.Equal(t, "de", settings.Locale)
	require.False(t, settings.Use24Hour)
	api.AssertExpectations(t)
}

func TestDisplayPreferencesFallbacks(t *testing.T) {
	for _, tc := range []struct {
		name     string
		user     *model.User
		userErr  *model.AppError
		prefs    []model.Preference
		prefErr  *model.AppError
		config   *model.Config
		locale   string
		military bool
	}{
		{name: "defaults", user: &model.User{}, locale: "en"},
		{name: "missing user", locale: "en"},
		{name: "server client locale", user: &model.User{}, config: &model.Config{LocalizationSettings: model.LocalizationSettings{DefaultClientLocale: model.NewPointer("ru")}}, locale: "ru"},
		{name: "preferences unavailable", user: &model.User{Locale: "ja"}, prefErr: &model.AppError{Message: "unavailable"}, locale: "ja"},
		{name: "user unavailable preserves clock", userErr: &model.AppError{Message: "unavailable"}, prefs: []model.Preference{{Category: model.PreferenceCategoryDisplaySettings, Name: model.PreferenceNameUseMilitaryTime, Value: "true"}}, locale: "en", military: true},
		{name: "invalid preference", user: &model.User{Locale: "fr"}, prefs: []model.Preference{{Category: model.PreferenceCategoryDisplaySettings, Name: model.PreferenceNameUseMilitaryTime, Value: "invalid"}}, locale: "fr"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api := &plugintest.API{}
			api.On("GetUser", "u").Return(tc.user, tc.userErr)
			api.On("GetPreferencesForUser", "u").Return(tc.prefs, tc.prefErr)
			api.On("GetConfig").Return(tc.config).Maybe()
			settings := NewDisplayPreferences(api).ForUser("u")
			require.Equal(t, tc.locale, settings.Locale)
			require.Equal(t, tc.military, settings.Use24Hour)
		})
	}
}
