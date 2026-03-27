package command

import (
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
)

// Interface defines the command handler contract.
type Interface interface {
	Register() error
	Execute(args *model.CommandArgs) (*model.CommandResponse, *model.AppError)
	BuildEphemeralList(args *model.CommandArgs) *model.CommandResponse
	UserDeleteMessage(userID, msgID string) (*types.ScheduledMessage, error)
	UserSendMessage(userID, msgID string) (*types.ScheduledMessage, error)
	ScheduleComposer(userID, channelID, rootPostID, message string, postAtUTC time.Time) (*types.ScheduledMessage, error)
}
