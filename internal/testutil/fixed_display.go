package testutil

import "github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"

// FixedUserDisplay returns fixed locale and clock mode for tests.
type FixedUserDisplay struct {
	Locale   string
	Military bool
}

var _ ports.UserDisplay = FixedUserDisplay{}

// LocaleAndMilitaryTime implements ports.UserDisplay.
func (f FixedUserDisplay) LocaleAndMilitaryTime(string) (string, bool) {
	loc := f.Locale
	if loc == "" {
		loc = "en"
	}
	return loc, f.Military
}
