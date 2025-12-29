package testutil

import (
	"fmt"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
)

// SchedKey builds a scheduled-message key for tests.
func SchedKey(id string) string {
	return fmt.Sprintf("%s%s", constants.SchedPrefix, id)
}

// IndexKey builds a user index key for tests.
func IndexKey(userID string) string {
	return fmt.Sprintf("%s%s", constants.UserIndexPrefix, userID)
}
