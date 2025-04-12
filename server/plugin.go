package main

import (
    "github.com/mattermost/mattermost/server/public/pluginapi"
    "github.com/mattermost/mattermost/server/public/plugin"
    "github.com/mattermost/mattermost/server/public/model"
		"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/command"
)

type Plugin struct {
    plugin.MattermostPlugin
		// configurationLock synchronizes access to the configuration.
		configurationLock sync.RWMutex
		// configuration is the active plugin configuration. Consult getConfiguration and
		// setConfiguration for usage.
		configuration *configuration
    API        *pluginapi.Client
    Scheduler  *scheduler.Scheduler
    Store      store.Store
    Command    *command.Handler
}

func (p *Plugin) OnActivate() error {
    p.API = pluginapi.NewClient(p.API, p.Driver)
    kvStore := store.NewKVStore(p.API.KV)
    sched := scheduler.New(p.API, kvStore)
    p.Scheduler = sched
    p.Store = kvStore
    p.Command = command.NewHandler(p.API, kvStore, sched)
    if err := p.API.Command.Register(p.Command.Definition()); err != nil {
        return err
    }
    go p.Scheduler.Start()
		p.API.LogInfo("Scheduled Messages plugin activated.")

    return nil
}

func (p *Plugin) OnDeactivate() error {
    p.Scheduler.Stop()
    return nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
    return p.Command.Execute(args)
}
















// package main
//
// import (
// 	// "encoding/json"
// 	// "fmt"
// 	// TODO: Need this?
// 	"net/http"
// 	// "strconv"
// 	// "strings"
// 	"sync"
// 	"time"
//
// 	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/command"
// 	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store/kvstore"
// 	"github.com/mattermost/mattermost/server/public/model"
// 	"github.com/mattermost/mattermost/server/public/plugin"
// 	"github.com/mattermost/mattermost/server/public/pluginapi"
// 	// "github.com/pkg/errors"
// )
//
// // ScheduledMessage holds info about a scheduled post.
// type ScheduledMessage struct {
// 	MessageID      string    `json:"message_id"`
// 	UserID         string    `json:"user_id"`
// 	ChannelID      string    `json:"channel_id"`
// 	PostAt         time.Time `json:"post_at"` // in UTC
// 	MessageContent string    `json:"message_content"`
// 	Timezone       string    `json:"time_zone"` // The user’s timezone at scheduling
// }
//
// // Plugin implements the Mattermost plugin interface.
// type Plugin struct {
// 	plugin.MattermostPlugin
//
// 	// kvstore is the client used to read/write KV records for this plugin.
// 	kvstore kvstore.KVStore
//
// 	// client is the Mattermost server API client.
// 	client *pluginapi.Client
//
// 	// commandClient is the client used to register and execute slash commands.
// 	commandClient command.Command
//
// 	// configurationLock synchronizes access to the configuration.
// 	configurationLock sync.RWMutex
//
// 	// configuration is the active plugin configuration. Consult getConfiguration and
// 	// setConfiguration for usage.
// 	configuration *configuration
//
// 	// stopSchedulerChan chan struct{} // signal to stop the scheduler on deactivate
//
// 	// Keep an in-memory map of scheduled tasks for quick scanning.
// 	// Key: messageID, Value: *ScheduledMessage
// 	// This is optional; we could rely fully on KV, but in-memory is more efficient for checking.
// 	scheduledTasks map[string]*ScheduledMessage
//
// 	// Protect concurrent access to scheduledTasks
// 	// tasksMutex sync.Mutex
//
// 	// Track the last used ID in memory. We'll store it in KV as well.
// 	lastID int64
// }
//
// func (p *Plugin) OnActivate() error {
// 	p.client = pluginapi.NewClient(p.API, p.Driver)
//
// 	p.kvstore = kvstore.NewKVStore(p.client)
//
// 	p.commandClient = command.NewCommandHandler(p.client)
//
// 	// Initialize our in-memory map
// 	p.scheduledTasks = make(map[string]*ScheduledMessage)
//
// 	lastID, err := p.kvstore.LoadLastIDFromKV()
// 	if err != nil {
// 		p.API.LogError("Failed to load last message ID from KV", "error", err.Error())
// 		// Non-fatal; we'll continue with lastID=0
// 	}
// 	p.lastID = lastID
//
// 	// // Load all existing scheduled tasks from KV
// 	// if err := p.loadAllScheduledTasks(); err != nil {
// 	//     p.API.LogError("Failed to load scheduled tasks from KV", "error", err.Error())
// 	//     // Also non-fatal
// 	// }
// 	//
// 	// // Start scheduler goroutine
// 	// p.stopSchedulerChan = make(chan struct{})
// 	// go p.runScheduler()
// 	//
// 	// p.API.LogInfo("Scheduled Messages plugin activated.", "LoadedTasksCount", len(p.scheduledTasks))
// 	// return nil
// 	return nil
// }
//
// func (p *Plugin) OnDeactivate() error {
// 	// // Signal the scheduler goroutine to stop
// 	// if p.stopSchedulerChan != nil {
// 	//     close(p.stopSchedulerChan)
// 	// }
//
// 	p.API.LogInfo("Scheduled Messages plugin deactivated")
// 	return nil
// }
//
// // This will execute the commands that were registered in the NewCommandHandler function.
// func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
// 	response, err := p.commandClient.Handle(args)
// 	if err != nil {
// 		return nil, model.NewAppError("ExecuteCommand", "plugin.command.execute_command.app_error", nil, err.Error(), http.StatusInternalServerError)
// 	}
// 	return response, nil
// }

// // loadLastIDFromKV retrieves the last used message ID counter from KV, or starts at 0 if missing.
// func (p *Plugin) loadLastIDFromKV() error {
//     data, appErr := p.API.KVGet("schedule_last_id")
//     if appErr != nil {
//         return appErr
//     }
//     if data == nil {
//         p.lastID = 0
//         return nil
//     }
//     val, err := strconv.ParseInt(string(data), 10, 64)
//     if err != nil {
//         return err
//     }
//     p.lastID = val
//     return nil
// }
//
// // storeLastIDToKV updates the lastID in the KV store.
// func (p *Plugin) storeLastIDToKV() error {
//     valStr := strconv.FormatInt(p.lastID, 10)
//     if appErr := p.API.KVSet("schedule_last_id", []byte(valStr)); appErr != nil {
//         return appErr
//     }
//     return nil
// }
//
// // loadAllScheduledTasks scans KV for keys with "schedmsg:" prefix and loads them into memory.
// func (p *Plugin) loadAllScheduledTasks() error {
//     var page int
//     pageSize := 100
//     for {
//         keys, appErr := p.API.KVList(page, pageSize)
//         if appErr != nil {
//             return appErr
//         }
//         if len(keys) == 0 {
//             break
//         }
//         for _, key := range keys {
//             if strings.HasPrefix(key, "schedmsg:") {
//                 data, appErr2 := p.API.KVGet(key)
//                 if appErr2 != nil {
//                     p.API.LogError("Failed to retrieve key from KV", "key", key, "error", appErr2.Error())
//                     continue
//                 }
//                 if data == nil {
//                     continue
//                 }
//                 var msg ScheduledMessage
//                 if err := json.Unmarshal(data, &msg); err != nil {
//                     p.API.LogError("Failed to unmarshal ScheduledMessage", "error", err.Error())
//                     continue
//                 }
//                 // Store in memory
//                 p.scheduledTasks[msg.MessageID] = &msg
//             }
//         }
//         if len(keys) < pageSize {
//             break
//         }
//         page++
//     }
//     return nil
// }
//
// // runScheduler polls every minute to post messages whose PostAt time has arrived.
// func (p *Plugin) runScheduler() {
//     ticker := time.NewTicker(1 * time.Minute)
//     defer ticker.Stop()
//
//     // Immediately check for overdue tasks at startup
//     p.checkAndPostDueMessages()
//
//     for {
//         select {
//         case <-ticker.C:
//             p.checkAndPostDueMessages()
//         case <-p.stopSchedulerChan:
//             return
//         }
//     }
// }
//
// // checkAndPostDueMessages finds any tasks that are due now/past, posts them, and removes them from store.
// func (p *Plugin) checkAndPostDueMessages() {
//     p.tasksMutex.Lock()
//     defer p.tasksMutex.Unlock()
//
//     now := time.Now().UTC()
//     var due []*ScheduledMessage
//
//     for _, msg := range p.scheduledTasks {
//         if !msg.PostAt.After(now) { // means PostAt <= now
//             due = append(due, msg)
//         }
//     }
//
//     for _, msg := range due {
//         p.postScheduledMessage(msg)
//     }
// }
//
// // postScheduledMessage posts as the scheduling user, then removes from KV.
// func (p *Plugin) postScheduledMessage(msg *ScheduledMessage) {
//     post := &model.Post{
//         ChannelId: msg.ChannelID,
//         Message:   msg.MessageContent,
//         UserId:    msg.UserID, // Post AS the scheduling user
//     }
//
//     createdPost, appErr := p.API.CreatePost(post)
//     if appErr != nil {
//         // If posting fails, log and notify the user (best effort).
//         p.API.LogError("failed to post scheduled message",
//             "MessageID", msg.MessageID,
//             "UserID", msg.UserID,
//             "ChannelID", msg.ChannelID,
//             "error", appErr.Error(),
//         )
//         p.notifyUserOfFailure(msg, appErr)
//     } else {
//         // Successfully posted
//         p.API.LogInfo("Posted scheduled message",
//             "MessageID", msg.MessageID,
//             "PostID", createdPost.Id,
//         )
//     }
//
//     // Remove from KV store
//     _ = p.removeScheduledMessageFromKV(msg)
// }
//
// // removeScheduledMessageFromKV removes the message from both main record and user index in KV.
// func (p *Plugin) removeScheduledMessageFromKV(msg *ScheduledMessage) error {
//     // Delete the main key
//     mainKey := "schedmsg:" + msg.MessageID
//     _ = p.API.KVDelete(mainKey)
//
//     // Remove from the user’s index list
//     indexKey := "user_sched_index:" + msg.UserID
//     data, appErr := p.API.KVGet(indexKey)
//     if appErr == nil && data != nil {
//         var index []string
//         if err := json.Unmarshal(data, &index); err == nil {
//             newIndex := make([]string, 0, len(index))
//             for _, id := range index {
//                 if id != msg.MessageID {
//                     newIndex = append(newIndex, id)
//                 }
//             }
//             newData, _ := json.Marshal(newIndex)
//             _ = p.API.KVSet(indexKey, newData)
//         }
//     }
//
//     // Remove from in-memory map
//     delete(p.scheduledTasks, msg.MessageID)
//     return nil
// }
//
// // notifyUserOfFailure tries to DM the user about the failure to post.
// func (p *Plugin) notifyUserOfFailure(msg *ScheduledMessage, appErr *model.AppError) {
//     // Attempt a direct channel to the user with themselves.
//     // In many Mattermost versions, a user cannot DM themselves, so this may fail.
//     // We do best effort. Alternatively, one might ephemeral-post in the same channel if possible.
//
//     ch, cErr := p.API.GetDirectChannel(msg.UserID, msg.UserID)
//     if cErr != nil || ch == nil {
//         p.API.LogError("Unable to create direct channel to notify user of failure",
//             "UserID", msg.UserID, "Error", cErr)
//         return
//     }
//
//     text := fmt.Sprintf("Your scheduled message (ID=%s) failed to post to channel [%s]. Error: %s",
//         msg.MessageID, msg.ChannelID, appErr.Error())
//
//     failPost := &model.Post{
//         ChannelId: ch.Id,
//         // Optionally post as the user or as the plugin's system user
//         UserId: msg.UserID,
//         Message: text,
//     }
//     _, err := p.API.CreatePost(failPost)
//     if err != nil {
//         p.API.LogError("Failed to notify user of scheduling failure", "err", err.Error())
//     }
// }
//
// // ExecuteCommand is the slash command entry point.
// func (p *Plugin) ExecuteCommand(_ *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
//     cmd := strings.TrimSpace(args.Command)
//     if !strings.HasPrefix(cmd, "/schedule") {
//         return &model.CommandResponse{}, nil
//     }
//
//     text := strings.TrimSpace(strings.TrimPrefix(cmd, "/schedule"))
//     if len(text) == 0 {
//         return p.helpResponse(), nil
//     }
//
//     if strings.HasPrefix(text, "list") {
//         return p.handleListCommand(args), nil
//     }
//     if strings.HasPrefix(text, "delete") {
//         return p.handleDeleteCommand(args, text), nil
//     }
//     // Otherwise, treat it as a new scheduling request
//     return p.handleScheduleCommand(args, text), nil
// }
//
// // helpResponse displays usage instructions.
// func (p *Plugin) helpResponse() *model.CommandResponse {
//     usage := "Usage:\n" +
//         "  /schedule at <time> [on <date>] message <text>\n" +
//         "  /schedule list\n" +
//         "  /schedule delete <id>"
//     return &model.CommandResponse{
//         ResponseType: model.CommandResponseTypeEphemeral,
//         Text:         usage,
//     }
// }
//
// // handleListCommand shows all pending scheduled messages for the current user.
// func (p *Plugin) handleListCommand(args *model.CommandArgs) *model.CommandResponse {
//     userID := args.UserId
//     indexKey := "user_sched_index:" + userID
//
//     data, appErr := p.API.KVGet(indexKey)
//     if appErr != nil {
//         return ephemeralError("Failed to retrieve scheduled messages.")
//     }
//     if data == nil {
//         return ephemeralMessage("You have no scheduled messages.")
//     }
//
//     var idList []string
//     if err := json.Unmarshal(data, &idList); err != nil {
//         return ephemeralError("Could not parse scheduled message list.")
//     }
//     if len(idList) == 0 {
//         return ephemeralMessage("You have no scheduled messages.")
//     }
//
//     // Gather them from in-memory map
//     p.tasksMutex.Lock()
//     var tasks []*ScheduledMessage
//     for _, id := range idList {
//         if m, ok := p.scheduledTasks[id]; ok {
//             tasks = append(tasks, m)
//         }
//     }
//     p.tasksMutex.Unlock()
//
//     if len(tasks) == 0 {
//         return ephemeralMessage("You have no scheduled messages.")
//     }
//
//     var sb strings.Builder
//     sb.WriteString("Your scheduled messages:\n")
//     for _, m := range tasks {
//         localTime := m.PostAt
//         loc, err := time.LoadLocation(m.Timezone)
//         if err == nil {
//             localTime = m.PostAt.In(loc)
//         }
//         sb.WriteString(fmt.Sprintf(
//             "- ID: %s, Time: %s, Channel: %s, Message: %q\n",
//             m.MessageID,
//             localTime.Format("2006-01-02 15:04"),
//             m.ChannelID,
//             m.MessageContent,
//         ))
//     }
//     return ephemeralMessage(sb.String())
// }
//
// func (p *Plugin) handleDeleteCommand(args *model.CommandArgs, text string) *model.CommandResponse {
//     userID := args.UserId
//     fields := strings.Fields(text) // e.g. ["delete","5"]
//     if len(fields) < 2 {
//         return ephemeralMessage("Please specify a message ID to delete: `/schedule delete <id>`.")
//     }
//     msgID := fields[1]
//
//     // Attempt deletion
//     if err := p.deleteScheduledMessage(userID, msgID); err != nil {
//         return ephemeralError(fmt.Sprintf("Failed to delete message %s: %v", msgID, err))
//     }
//     return ephemeralMessage(fmt.Sprintf("Deleted scheduled message %s.", msgID))
// }
//
// // deleteScheduledMessage is the KV + memory removal, ensuring the user owns the message.
// func (p *Plugin) deleteScheduledMessage(userID, messageID string) error {
//     key := "schedmsg:" + messageID
//     data, appErr := p.API.KVGet(key)
//     if appErr != nil {
//         return appErr
//     }
//     if data == nil {
//         return errors.New("not found or already deleted")
//     }
//
//     var msg ScheduledMessage
//     if err := json.Unmarshal(data, &msg); err != nil {
//         return err
//     }
//     if msg.UserID != userID {
//         return errors.New("not authorized or belongs to a different user")
//     }
//
//     if err := p.API.KVDelete(key); err != nil {
//         return err
//     }
//     // Remove from user index
//     indexKey := "user_sched_index:" + userID
//     idxData, err := p.API.KVGet(indexKey)
//     if err == nil && idxData != nil {
//         var idx []string
//         if unErr := json.Unmarshal(idxData, &idx); unErr == nil {
//             out := make([]string, 0, len(idx))
//             for _, id := range idx {
//                 if id != messageID {
//                     out = append(out, id)
//                 }
//             }
//             newBytes, _ := json.Marshal(out)
//             _ = p.API.KVSet(indexKey, newBytes)
//         }
//     }
//
//     p.tasksMutex.Lock()
//     delete(p.scheduledTasks, messageID)
//     p.tasksMutex.Unlock()
//     return nil
// }
//
// // handleScheduleCommand creates a new scheduled message record.
// func (p *Plugin) handleScheduleCommand(args *model.CommandArgs, text string) *model.CommandResponse {
//     req, err := p.parseScheduleText(text)
//     if err != nil {
//         return ephemeralError("Error parsing schedule command: " + err.Error())
//     }
//
//     userID := args.UserId
//     channelID := args.ChannelId
//     userTz := p.getUserTimezone(userID)
//
//     loc, locErr := time.LoadLocation(userTz)
//     if locErr != nil {
//         loc = time.UTC
//     }
//
//     // The local "now" for the user
//     nowLocal := time.Now().In(loc)
//
//     // Parse the time-of-day
//     parsedTime, timeErr := p.tryParseTime(req.timeOfDay, loc)
//     if timeErr != nil {
//         return ephemeralError(fmt.Sprintf("Invalid time format: %s", req.timeOfDay))
//     }
//
//     // Determine the date
//     var year, month, day int
//     if req.date != "" {
//         dt, dErr := time.ParseInLocation("2006-01-02", req.date, loc)
//         if dErr != nil {
//             return ephemeralError("Invalid date format (use YYYY-MM-DD).")
//         }
//         year, month, day = dt.Year(), int(dt.Month()), dt.Day()
//     } else {
//         // If no 'on' date, decide if it's for today or tomorrow
//         year, month, day = nowLocal.Year(), int(nowLocal.Month()), nowLocal.Day()
//         candidate := time.Date(year, time.Month(month), day, parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)
//         if !candidate.After(nowLocal) {
//             // schedule for tomorrow
//             tomorrow := nowLocal.Add(24 * time.Hour)
//             year, month, day = tomorrow.Year(), int(tomorrow.Month()), tomorrow.Day()
//         }
//     }
//
//     postTimeLocal := time.Date(year, time.Month(month), day, parsedTime.Hour(), parsedTime.Minute(), 0, 0, loc)
//     if postTimeLocal.Before(nowLocal) {
//         return ephemeralError("You specified a time in the past.")
//     }
//
//     // Convert to UTC
//     postTimeUTC := postTimeLocal.UTC()
//
//     // Generate a new message ID
//     p.tasksMutex.Lock()
//     p.lastID++
//     msgID := fmt.Sprintf("%d", p.lastID)
//     _ = p.storeLastIDToKV()
//     p.tasksMutex.Unlock()
//
//     schedMsg := &ScheduledMessage{
//         MessageID:      msgID,
//         UserID:         userID,
//         ChannelID:      channelID,
//         PostAt:         postTimeUTC,
//         MessageContent: req.messageContent,
//         Timezone:       userTz,
//     }
//
//     // Persist to KV
//     if err2 := p.storeScheduledMessage(schedMsg); err2 != nil {
//         return ephemeralError(fmt.Sprintf("Failed to store scheduled message: %v", err2))
//     }
//
//     // Add to in-memory map
//     p.tasksMutex.Lock()
//     p.scheduledTasks[msgID] = schedMsg
//     p.tasksMutex.Unlock()
//
//     confirmation := fmt.Sprintf("✅ Scheduled message #%s for %s (%s).",
//         msgID, postTimeLocal.Format("2006-01-02 15:04"), userTz)
//     return ephemeralMessage(confirmation)
// }
//
// // scheduleRequest holds the parsed data from command text.
// type scheduleRequest struct {
//     timeOfDay      string
//     date           string
//     messageContent string
// }
//
// // parseScheduleText looks for "at <time> [on <date>] message <text>"
// func (p *Plugin) parseScheduleText(text string) (*scheduleRequest, error) {
//     lower := strings.ToLower(text)
//     atIdx := strings.Index(lower, " at ")
//     if atIdx < 0 {
//         return nil, fmt.Errorf("missing 'at' keyword (syntax: at <time> ...)")
//     }
//
//     msgIdx := strings.Index(lower, " message ")
//     if msgIdx < 0 {
//         return nil, fmt.Errorf("missing 'message' keyword")
//     }
//
//     timePart := strings.TrimSpace(text[atIdx+4:]) // everything after " at "
//     onIdx := strings.Index(strings.ToLower(timePart), " on ")
//
//     var datePart string
//     if onIdx >= 0 {
//         // we have " on "
//         dateAndAfter := timePart[onIdx+4:]
//         timePart = strings.TrimSpace(timePart[:onIdx])
//         // the message keyword is presumably in dateAndAfter, but we rely on msgIdx globally
//         // We'll not re-check that here. We'll parse date from dateAndAfter up to " message "?
//         // Actually, we just read everything up to the end, then the parse from the main text for message is simpler.
//
//         // but we can do a simpler approach:
//         // datePart is everything from the start of dateAndAfter up to the substring " message "
//         // but we've already found msgIdx in the entire command. Let's do this more robustly:
//         // We'll just assume dateAndAfter is the date, because " message " is in the global text.
//         // If user typed something weird, we might fail. For brevity, let's do a substring approach:
//
//         // In a more robust parser, we'd match tokens carefully. For now, let's do a naive slice.
//         possibleMsgIdx := strings.Index(strings.ToLower(dateAndAfter), " message ")
//         if possibleMsgIdx < 0 {
//             // means we had "on" but no "message" afterwards
//             return nil, fmt.Errorf("missing 'message' keyword after 'on'")
//         }
//         datePart = strings.TrimSpace(dateAndAfter[:possibleMsgIdx])
//     }
//
//     // The message content is everything after " message "
//     messagePart := strings.TrimSpace(text[msgIdx+9:])
//     if messagePart == "" {
//         return nil, fmt.Errorf("missing message content")
//     }
//
//     req := &scheduleRequest{
//         timeOfDay:      timePart,
//         date:           datePart,
//         messageContent: messagePart,
//     }
//     return req, nil
// }
//
// // storeScheduledMessage writes a ScheduledMessage to KV and updates the user's index
// func (p *Plugin) storeScheduledMessage(msg *ScheduledMessage) error {
//     key := "schedmsg:" + msg.MessageID
//     data, err := json.Marshal(msg)
//     if err != nil {
//         return err
//     }
//     if appErr := p.API.KVSet(key, data); appErr != nil {
//         return appErr
//     }
//
//     // Update user index
//     indexKey := "user_sched_index:" + msg.UserID
//     indexData, appErr := p.API.KVGet(indexKey)
//     var ids []string
//     if indexData != nil && appErr == nil {
//         _ = json.Unmarshal(indexData, &ids)
//     }
//     found := false
//     for _, existing := range ids {
//         if existing == msg.MessageID {
//             found = true
//             break
//         }
//     }
//     if !found {
//         ids = append(ids, msg.MessageID)
//     }
//     newIndex, _ := json.Marshal(ids)
//     if err2 := p.API.KVSet(indexKey, newIndex); err2 != nil {
//         return err2
//     }
//     return nil
// }
//
// // tryParseTime attempts multiple time formats (24-hr, 12-hr, etc.)
// func (p *Plugin) tryParseTime(timeStr string, loc *time.Location) (time.Time, error) {
//     layouts := []string{"15:04", "3:04PM", "3:04pm", "3pm"}
//     var parsed time.Time
//     var err error
//     for _, layout := range layouts {
//         parsed, err = time.ParseInLocation(layout, timeStr, loc)
//         if err == nil {
//             return parsed, nil
//         }
//     }
//     return time.Time{}, err
// }
//
// // getUserTimezone attempts to retrieve the user's timezone setting from their props.
// func (p *Plugin) getUserTimezone(userID string) string {
//     user, appErr := p.API.GetUser(userID)
//     if appErr != nil {
//         return "UTC"
//     }
//     if tzProps, ok := user.Props["timezone"].(map[string]interface{}); ok {
//         auto, aok := tzProps["automaticTimezone"].(string)
//         man, mok := tzProps["manualTimezone"].(string)
//         if aok && auto != "" {
//             return auto
//         }
//         if mok && man != "" {
//             return man
//         }
//     }
//     return "UTC"
// }
//
// // ephemeralMessage wraps text in an ephemeral command response.
// func ephemeralMessage(text string) *model.CommandResponse {
//     return &model.CommandResponse{
//         ResponseType: model.CommandResponseTypeEphemeral,
//         Text:         text,
//     }
// }
//
// // ephemeralError is like ephemeralMessage, but indicates an error in text.
// func ephemeralError(text string) *model.CommandResponse {
//     return ephemeralMessage("❌ " + text)
// }
//
// var _ plugin.Plugin = (*Plugin)(nil)
//
// // main is required to build the plugin.
// func main() {
//     plugin.ClientMain(&Plugin{})
// }
//
