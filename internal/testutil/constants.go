package testutil

import (
	"fmt"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
)

func SchedKey(id string) string {
	return fmt.Sprintf("%s%s", constants.SchedPrefix, id)
}

func IndexKey(userID string) string {
	return fmt.Sprintf("%s%s", constants.UserIndexPrefix, userID)
}
