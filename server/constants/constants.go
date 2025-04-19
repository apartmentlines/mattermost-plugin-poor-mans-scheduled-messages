package constants

const (
	// SchedPrefix is the prefix used for scheduled message keys in the KV store.
	SchedPrefix = "schedmsg:"
	// UserIndexPrefix is the prefix used for user message index keys in the KV store.
	UserIndexPrefix = "user_sched_index:"
	// MaxUserMessages is a common limit used in tests involving user message counts.
	MaxUserMessages = 1000
	AssetsDir       = "assets"

	// Bot Configuration
	ProfileImageFilename = "profile.png"

	// Command Strings & Autocomplete
	CommandTrigger            = "schedule"
	CommandDisplayName        = "Schedule"
	CommandDescription        = "Send messages at a future time."
	SubcommandHelp            = "help"
	SubcommandList            = "list"
	SubcommandAt              = "at"
	AutocompleteDesc          = "Schedule messages to be sent later"
	AutocompleteHint          = "[subcommand]"
	AutocompleteAtHint        = "<time> [on <date>] message <text>"
	AutocompleteAtDesc        = "Schedule a new message"
	AutocompleteAtArgTimeName = "Time"
	AutocompleteAtArgTimeHint = "Time to send the message, e.g. 3:15PM, 3pm"
	AutocompleteAtArgDateName = "Date"
	AutocompleteAtArgDateHint = "(Optional) Date to send the message, e.g. 2026-01-01"
	AutocompleteAtArgMsgName  = "Message"
	AutocompleteAtArgMsgHint  = "The message content"
	AutocompleteListHint      = ""
	AutocompleteListDesc      = "List your scheduled messages"
	AutocompleteHelpHint      = ""
	AutocompleteHelpDesc      = "Show help text"
	EmptyScheduleMessage      = "Trying to schedule a message? Use %s for instructions."

	// Parser Errors
	ParserErrInvalidFormat     = "invalid format. Use: `at <time> [on <date>] message <your message text>`"
	ParserErrInvalidDateFormat = "invalid date format specified: '%s'. Use YYYY-MM-DD, day name (e.g., 'tuesday', 'fri'), or short date (e.g., '3jan', '25dec')"
	ParserErrUnknownDateFormat = "unknown date format detected"

	// API & HTTP
	HTTPHeaderMattermostUserID = "Mattermost-User-ID"

	// Formatting & Display Strings
	TimeLayout                = "Jan 2, 2006 3:04 PM"
	EmojiSuccess              = "✅"
	EmojiError                = "❌"
	UnknownChannelPlaceholder = "N/A"
	EmptyListMessage          = "You have no scheduled messages."
	ListHeader                = "### Scheduled Messages"

	// Time & Scheduling
	DefaultTimezone         = "UTC"
	DateParseLayoutYYYYMMDD = "2006-01-02"

	// File Paths
	HelpFilename = "help.md"

	// Pagination/Limits
	DefaultPage                  = 0
	DefaultChannelMembersPerPage = 100
)

// TimeParseLayouts defines the acceptable formats for parsing time strings.
var TimeParseLayouts = []string{"15:04", "3:04pm", "3:04PM", "3pm", "3PM"}
