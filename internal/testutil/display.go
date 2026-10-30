package testutil

import "github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"

// DisplayPreferences is a function adapter for tests using live display settings.
type DisplayPreferences func(string) types.DisplaySettings

// ForUser loads the requested recipient's display settings.
func (f DisplayPreferences) ForUser(userID string) types.DisplaySettings { return f(userID) }
