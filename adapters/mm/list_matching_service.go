package mm

import (
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type listMatchingService struct{}

// NewListMatchingService returns a ListMatchingService backed by pluginapi helpers.
func NewListMatchingService() ports.ListMatchingService {
	return listMatchingService{}
}

func (listMatchingService) WithPrefix(p string) pluginapi.ListKeysOption {
	return pluginapi.WithPrefix(p)
}
