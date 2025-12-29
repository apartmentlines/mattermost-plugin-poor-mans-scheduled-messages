package ports

import (
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/clock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/types"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

// Logger defines the logging interface used by the plugin.
type Logger interface {
	Error(msg string, keyvals ...any)
	Warn(msg string, keyvals ...any)
	Info(msg string, keyvals ...any)
	Debug(msg string, keyvals ...any)
}

// Clock aliases the clock interface for shared use.
type Clock = clock.Clock

// PostService abstracts Mattermost post operations.
type PostService interface {
	CreatePost(post *model.Post) error
	DM(botID, userID string, post *model.Post) error
	UpdateEphemeralPost(userID string, post *model.Post)
	SendEphemeralPost(userID string, post *model.Post)
}

// ChannelInfo holds channel metadata for formatting.
type ChannelInfo struct {
	ChannelID   string
	ChannelType model.ChannelType
	ChannelLink string
	TeamName    string
}

// ChannelService provides channel metadata and formatting helpers.
type ChannelService interface {
	GetInfoOrUnknown(channelID string) *ChannelInfo
	MakeChannelLink(info *ChannelInfo) string
}

// ChannelDataService provides channel data access.
type ChannelDataService interface {
	Get(channelID string) (*model.Channel, error)
	ListMembers(channelID string, page, perPage int) ([]*model.ChannelMember, error)
}

// TeamService provides team data access.
type TeamService interface {
	Get(teamID string) (*model.Team, error)
}

// SlashCommandService registers slash commands.
type SlashCommandService interface {
	Register(cmd *model.Command) error
}

// UserService fetches user data.
type UserService interface {
	Get(userID string) (*model.User, error)
}

// KVService abstracts key-value storage.
type KVService interface {
	Get(key string, val any) error
	Set(string, any, ...pluginapi.KVSetOption) (bool, error)
	Delete(key string) error
	ListKeys(page, perPage int, opts ...pluginapi.ListKeysOption) ([]string, error)
}

// BotService manages bot accounts.
type BotService interface {
	EnsureBot(bot *model.Bot, profileImagePath ...pluginapi.EnsureBotOption) (string, error)
}

// BotProfileImageService configures bot profile images.
type BotProfileImageService interface {
	ProfileImagePath(path string) pluginapi.EnsureBotOption
}

// ListMatchingService provides list-key filter options.
type ListMatchingService interface {
	WithPrefix(prefix string) pluginapi.ListKeysOption
}

// Store persists scheduled messages.
type Store interface {
	SaveScheduledMessage(userID string, msg *types.ScheduledMessage) error
	DeleteScheduledMessage(userID string, msgID string) error
	CleanupMessageFromUserIndex(userID string, msgID string) error
	GetScheduledMessage(msgID string) (*types.ScheduledMessage, error)
	ListScheduledMessages() ([]*types.ScheduledMessage, error)
	ListUserMessageIDs(userID string) ([]string, error)
	GenerateMessageID() string
}

// Scheduler manages scheduled message delivery.
type Scheduler interface {
	Start()
	Stop()
	SendNow(msg *types.ScheduledMessage) error
}

// ListService builds scheduled message lists.
type ListService interface {
	Build(userID string) *model.CommandResponse
}

// ScheduleService schedules new messages.
type ScheduleService interface {
	Build(args *model.CommandArgs, text string) *model.CommandResponse
}
