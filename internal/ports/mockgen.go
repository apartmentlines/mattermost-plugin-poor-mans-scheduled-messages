// Code generation directives for gomock. Run "make mocks".

package ports

//go:generate mockgen -destination=../../adapters/mock/post_mock.go -package=mock github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports PostService
//go:generate mockgen -destination=../../adapters/mock/channel_mock.go -package=mock github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports ChannelService
//go:generate mockgen -destination=../../adapters/mock/kv_mock.go -package=mock github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports KVService
//go:generate mockgen -destination=../../adapters/mock/bot_mock.go -package=mock github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports BotService
//go:generate mockgen -destination=../../adapters/mock/channeldata_mock.go -package=mock github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports ChannelDataService
//go:generate mockgen -destination=../../adapters/mock/team_mock.go -package=mock github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports TeamService
//go:generate mockgen -destination=../../adapters/mock/slash_mock.go -package=mock github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports SlashCommandService
//go:generate mockgen -destination=../../adapters/mock/user_mock.go -package=mock github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports UserService
//go:generate mockgen -destination=../../adapters/mock/store_mock.go -package=mock github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports Store
//go:generate mockgen -destination=../../adapters/mock/scheduler_mock.go -package=mock github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports Scheduler
//go:generate mockgen -destination=../../adapters/mock/list_service_mock.go -package=mock github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports ListService
//go:generate mockgen -destination=../../adapters/mock/schedule_service_mock.go -package=mock github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports ScheduleService
