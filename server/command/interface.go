package command

import (
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
)

type Interface interface {
	Register() error
	Execute(args *model.CommandArgs) (*model.CommandResponse, *model.AppError)
	BuildEphemeralList(args *model.CommandArgs) *model.CommandResponse
	UserDeleteMessage(userID, msgID string) (*types.ScheduledMessage, error)
}
