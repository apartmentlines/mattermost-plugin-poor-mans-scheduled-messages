package ports

import (
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/clock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type Logger interface {
	Error(msg string, keyvals ...any)
	Warn(msg string, keyvals ...any)
	Info(msg string, keyvals ...any)
	Debug(msg string, keyvals ...any)
}

type Clock = clock.Clock

type PostService interface {
	CreatePost(post *model.Post) error
	DM(botID, userID string, post *model.Post) error
	UpdateEphemeralPost(userID string, post *model.Post)
	SendEphemeralPost(userID string, post *model.Post)
}

type ChannelInfo struct {
	ChannelID   string
	ChannelType model.ChannelType
	ChannelLink string
	TeamName    string
}

type ChannelService interface {
	GetInfoOrUnknown(channelID string) *ChannelInfo
	MakeChannelLink(info *ChannelInfo) string
}

type ChannelDataService interface {
	Get(channelID string) (*model.Channel, error)
	ListMembers(channelID string, page, perPage int) ([]*model.ChannelMember, error)
}

type TeamService interface {
	Get(teamID string) (*model.Team, error)
}

type SlashCommandService interface {
	Register(cmd *model.Command) error
}

type UserService interface {
	Get(userID string) (*model.User, error)
}

type KVService interface {
	Get(key string, val any) error
	Set(string, any, ...pluginapi.KVSetOption) (bool, error)
	Delete(key string) error
	ListKeys(page, perPage int, opts ...pluginapi.ListKeysOption) ([]string, error)
}

type BotService interface {
	EnsureBot(bot *model.Bot, profileImagePath ...pluginapi.EnsureBotOption) (string, error)
}

type BotProfileImageService interface {
	ProfileImagePath(path string) pluginapi.EnsureBotOption
}

type ListMatchingService interface {
	WithPrefix(prefix string) pluginapi.ListKeysOption
}

type Store interface {
	SaveScheduledMessage(userID string, msg *types.ScheduledMessage) error
	DeleteScheduledMessage(userID string, msgID string) error
	CleanupMessageFromUserIndex(userID string, msgID string) error
	GetScheduledMessage(msgID string) (*types.ScheduledMessage, error)
	ListScheduledMessages() ([]*types.ScheduledMessage, error)
	ListUserMessageIDs(userID string) ([]string, error)
	GenerateMessageID() string
}

type Scheduler interface {
	Start()
	Stop()
}

type ListService interface {
	Build(userID string) *model.CommandResponse
}

type ScheduleService interface {
	Build(args *model.CommandArgs, text string) *model.CommandResponse
}
