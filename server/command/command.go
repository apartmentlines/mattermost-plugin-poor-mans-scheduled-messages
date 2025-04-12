package command
//
// import (
// 	"fmt"
// 	"strings"
//
// 	"github.com/mattermost/mattermost/server/public/model"
// 	"github.com/mattermost/mattermost/server/public/pluginapi"
// )
//
// type Handler struct {
// 	client *pluginapi.Client
// }
//
// type Command interface {
// 	Handle(args *model.CommandArgs) (*model.CommandResponse, error)
// 	executeHelloCommand(args *model.CommandArgs) *model.CommandResponse
// }
//
// const scheduleCommandTrigger = "schedule"
//
// // Register all your slash commands in the NewCommandHandler function.
// func NewCommandHandler(client *pluginapi.Client) Command {
// 	err := client.SlashCommand.Register(&model.Command{
// 		Trigger:          scheduleCommandTrigger,
// 		DisplayName:      "Schedule Message",
// 		Description:      "Schedule messages to be posted in the future.",
// 		AutoComplete:     true,
// 		AutoCompleteDesc: "Subcommands: at <time> [on <date>] message <text>, list, delete <id>",
// 		AutoCompleteHint: "[command]",
// 		AutocompleteData: model.NewAutocompleteData(scheduleCommandTrigger, "at <time> [on <date>] message <text>", "Syntax to schedule a message"),
// 	})
// 	if err != nil {
// 		client.Log.Error("Failed to register command", "error", err)
// 	}
// 	return &Handler{
// 		client: client,
// 	}
// }
//
// // ExecuteCommand hook calls this method to execute the commands that were registered in the NewCommandHandler function.
// func (c *Handler) Handle(args *model.CommandArgs) (*model.CommandResponse, error) {
// 	trigger := strings.TrimPrefix(strings.Fields(args.Command)[0], "/")
// 	switch trigger {
// 	case scheduleCommandTrigger:
// 		return c.executeHelloCommand(args), nil
// 	default:
// 		return &model.CommandResponse{
// 			ResponseType: model.CommandResponseTypeEphemeral,
// 			Text:         fmt.Sprintf("Unknown command: %s", args.Command),
// 		}, nil
// 	}
// }
//
// // ExecuteCommand is the slash command entry point.
// func (c *Handler) executeHelloCommand(args *model.CommandArgs) *model.CommandResponse {
// 	cmd := strings.TrimSpace(args.Command)
// 	if !strings.HasPrefix(cmd, "/schedule") {
// 		// TODO: Does this even need to be here?
// 		return &model.CommandResponse{}
// 	}
//
// 	text := strings.TrimSpace(strings.TrimPrefix(cmd, "/schedule"))
// 	if len(text) == 0 {
// 		return c.helpResponse()
// 	}
//
// 	if strings.HasPrefix(text, "list") {
// 		// return c.handleListCommand(args), nil
// 		return c.notImplemented()
// 	}
// 	if strings.HasPrefix(text, "delete") {
// 		// return c.handleDeleteCommand(args, text), nil
// 		return c.notImplemented()
// 	}
// 	// Otherwise, treat it as a new scheduling request
// 	// return p.handleScheduleCommand(args, text)
// 	return &model.CommandResponse{
// 		ResponseType: model.CommandResponseTypeEphemeral,
// 		Text:         fmt.Sprintf("You tried to schedule with: %s", text),
// 	}
// }
//
// // helpResponse displays usage instructions.
// func (c *Handler) helpResponse() *model.CommandResponse {
// 	usage := "Usage:\n" +
// 		"  /schedule at <time> [on <date>] message <text>\n" +
// 		"  /schedule list\n" +
// 		"  /schedule delete <id>"
// 	return &model.CommandResponse{
// 		ResponseType: model.CommandResponseTypeEphemeral,
// 		Text:         usage,
// 	}
// }
//
// func (c *Handler) notImplemented() *model.CommandResponse {
// 	usage := "Not implemented yet"
// 	return &model.CommandResponse{
// 		ResponseType: model.CommandResponseTypeEphemeral,
// 		Text:         usage,
// 	}
// }
