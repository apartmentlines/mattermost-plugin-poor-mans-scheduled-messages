// Package constants defines shared plugin constants.
package constants

const (
	// SchedPrefix is the prefix used for scheduled message keys in the KV store.
	SchedPrefix = "schedmsg:"
	// UserIndexPrefix is the prefix used for user message index keys in the KV store.
	UserIndexPrefix = "user_sched_index:"
	// MaxUserMessages is a common limit used in tests involving user message counts.
	MaxUserMessages = 1000
	// MaxMessageBytes is the maximum message size in bytes.
	MaxMessageBytes = 50 * 1024
	// AssetsDir is the plugin assets directory name.
	AssetsDir = "assets"

	// Bot Configuration

	// ProfileImageFilename is the bot profile image filename.
	ProfileImageFilename = "profile.png"

	// Command Strings & Autocomplete

	// CommandTrigger is the slash command trigger word.
	CommandTrigger = "schedule"
	// CommandDisplayName is the display name for the command.
	CommandDisplayName = "Schedule"
	// CommandDescription is the user-visible description for the command.
	CommandDescription = "Send messages at a future time."
	// SubcommandHelp is the help subcommand keyword.
	SubcommandHelp = "help"
	// SubcommandList is the list subcommand keyword.
	SubcommandList = "list"
	// SubcommandAt is the schedule subcommand keyword.
	SubcommandAt = "at"
	// AutocompleteDesc is the description used in autocomplete.
	AutocompleteDesc = "Schedule messages to be sent later"
	// AutocompleteHint is the hint used in autocomplete.
	AutocompleteHint = "[subcommand]"
	// AutocompleteAtHint is the hint for the schedule subcommand.
	AutocompleteAtHint = "<time> [on <date>] message <text>"
	// AutocompleteAtDesc describes the schedule subcommand.
	AutocompleteAtDesc = "Schedule a new message"
	// AutocompleteAtArgTimeName is the name of the time argument.
	AutocompleteAtArgTimeName = "Time"
	// AutocompleteAtArgTimeHint is the hint for the time argument.
	AutocompleteAtArgTimeHint = "Time to send the message, e.g. 3:15PM, 3pm"
	// AutocompleteAtArgDateName is the name of the date argument.
	AutocompleteAtArgDateName = "Date"
	// AutocompleteAtArgDateHint is the hint for the date argument.
	AutocompleteAtArgDateHint = "(Optional) Date to send the message, e.g. 2026-01-01"
	// AutocompleteAtArgMsgName is the name of the message argument.
	AutocompleteAtArgMsgName = "Message"
	// AutocompleteAtArgMsgHint is the hint for the message argument.
	AutocompleteAtArgMsgHint = "The message content"
	// AutocompleteListHint is the hint for the list subcommand.
	AutocompleteListHint = ""
	// AutocompleteListDesc describes the list subcommand.
	AutocompleteListDesc = "List your scheduled messages"
	// AutocompleteHelpHint is the hint for the help subcommand.
	AutocompleteHelpHint = ""
	// AutocompleteHelpDesc describes the help subcommand.
	AutocompleteHelpDesc = "Show help text"
	// EmptyScheduleMessage is shown when a schedule command is empty.
	EmptyScheduleMessage = "Trying to schedule a message? Use %s for instructions."

	// Parser Errors

	// ParserErrInvalidFormat is returned for invalid command formats.
	ParserErrInvalidFormat = "invalid format. Use: `at <time> [on <date>] message <your message text>`"
	// ParserErrInvalidDateFormat is returned for invalid date inputs.
	ParserErrInvalidDateFormat = "invalid date format specified: '%s'. Use YYYY-MM-DD, day name (e.g., 'tuesday', 'fri'), or short date (e.g., '3jan', '25dec')"
	// ParserErrUnknownDateFormat is returned for unknown date formats.
	ParserErrUnknownDateFormat = "unknown date format detected"

	// API & HTTP

	// HTTPHeaderMattermostUserID is the header containing the Mattermost user ID.
	HTTPHeaderMattermostUserID = "Mattermost-User-ID"

	// Formatting & Display Strings

	// TimeLayout is the time format for user-facing messages.
	TimeLayout = "Jan 2, 2006 3:04 PM"
	// EmojiSuccess is the success indicator emoji.
	EmojiSuccess = "✅"
	// EmojiError is the error indicator emoji.
	EmojiError = "❌"
	// UnknownChannelPlaceholder is used when channel info is unavailable.
	UnknownChannelPlaceholder = "N/A"
	// EmptyListMessage is shown when no scheduled messages exist.
	EmptyListMessage = "You have no scheduled messages."
	// ListHeader is the heading for the list response.
	ListHeader = "### Scheduled Messages"

	// Time & Scheduling

	// DefaultTimezone is the fallback timezone.
	DefaultTimezone = "UTC"
	// DateParseLayoutYYYYMMDD is the YYYY-MM-DD date layout.
	DateParseLayoutYYYYMMDD = "2006-01-02"

	// File Paths

	// HelpFilename is the help text filename.
	HelpFilename = "help.md"

	// Pagination/Limits

	// DefaultPage is the default pagination page.
	DefaultPage = 0
	// DefaultChannelMembersPerPage is the default channel members page size.
	DefaultChannelMembersPerPage = 100
	// MaxFetchScheduledMessages is the maximum number of scheduled messages to fetch.
	MaxFetchScheduledMessages = 10000
)

// TimeParseLayouts defines the acceptable formats for parsing time strings.
var TimeParseLayouts = []string{"15:04", "3:04pm", "3:04PM", "3pm", "3PM"}
