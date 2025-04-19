package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/adapters/mm"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/bot"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/channel"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/clock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/command"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/scheduler"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type ClientFactory func(api plugin.API, drv plugin.Driver) *pluginapi.Client

type ClockFactory func() ports.Clock

type BotEnsurer func(ports.BotService, ports.BotProfileImageService) (string, error)

type AppBuilder interface {
	NewChannel(cli *pluginapi.Client) *channel.Channel
	NewStore(cli *pluginapi.Client, maxUserMessages int) store.Store
	NewScheduler(cli *pluginapi.Client, st store.Store, ch ports.ChannelService, botID string, clk ports.Clock) *scheduler.Scheduler
	NewCommandHandler(cli *pluginapi.Client, st store.Store, sched *scheduler.Scheduler, ch ports.ChannelService, maxUserMessages int, clk ports.Clock, help string) *command.Handler
}

type prodBuilder struct{}

func (prodBuilder) NewChannel(cli *pluginapi.Client) *channel.Channel {
	return channel.New(&cli.Log, &cli.Channel, &cli.Team, &cli.User)
}

func (prodBuilder) NewStore(cli *pluginapi.Client, maxUserMessages int) store.Store {
	return store.NewKVStore(&cli.Log, &cli.KV, mm.ListMatchingService{}, maxUserMessages)
}

func (prodBuilder) NewScheduler(cli *pluginapi.Client, st store.Store, ch ports.ChannelService, botID string, clk ports.Clock) *scheduler.Scheduler {
	return scheduler.New(&cli.Log, &cli.Post, st, ch, botID, clk)
}

func (prodBuilder) NewCommandHandler(cli *pluginapi.Client, st store.Store, sched *scheduler.Scheduler, ch ports.ChannelService, maxUserMessages int, clk ports.Clock, help string) *command.Handler {
	return command.NewHandler(&cli.Log, &cli.SlashCommand, &cli.User, st, sched, ch, maxUserMessages, clk, help)
}

type Plugin struct {
	plugin.MattermostPlugin
	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex
	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration          *configuration
	client                 *pluginapi.Client
	BotID                  string
	Scheduler              *scheduler.Scheduler
	Store                  store.Store
	Channel                ports.ChannelService
	Command                command.Interface
	defaultMaxUserMessages int
	helpText               string
	logger                 ports.Logger
	poster                 ports.PostService
}

func (p *Plugin) loadHelpText(text string) (string, error) {
	if text != "" {
		return text, nil
	}
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return "", fmt.Errorf("failed to get bundle path: %w", err)
	}
	helpFilePath := filepath.Join(bundlePath, "assets", "help.md")
	helpBytes, err := os.ReadFile(helpFilePath)
	if err != nil {
		return "", fmt.Errorf("failed to read help file %s: %w", helpFilePath, err)
	}
	return string(helpBytes), nil
}

func (p *Plugin) OnActivate() error {
	return p.OnActivateWith(pluginapi.NewClient, clock.NewReal, nil, bot.EnsureBot, "")
}

func (p *Plugin) OnActivateWith(
	clientFactory ClientFactory,
	clockFactory ClockFactory,
	builder AppBuilder,
	ensureBot BotEnsurer,
	help string,
) error {
	p.client = clientFactory(p.API, p.Driver)
	var helpText string
	var helpErr error
	if helpText, helpErr = p.loadHelpText(help); helpErr != nil {
		p.API.LogError("Plugin activation failed: could not load help text.", "error", helpErr.Error())
		return helpErr
	}
	p.helpText = helpText
	botID, botErr := ensureBot(&p.client.Bot, mm.BotProfileImageServiceWrapper{})
	if botErr != nil {
		p.API.LogError("Plugin activation failed: could not ensure bot.", "error", botErr.Error())
		return botErr
	}
	if builder == nil {
		builder = prodBuilder{}
	}
	clk := clockFactory()
	if initErr := p.initialize(botID, clk, builder); initErr != nil {
		p.API.LogError("Plugin activation failed: could not initialize dependencies.", "error", initErr.Error())
		return initErr
	}
	p.API.LogInfo("Scheduled Messages plugin activated.")
	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.Scheduler.Stop()
	return nil
}

func (p *Plugin) initialize(botID string, clk ports.Clock, builder AppBuilder) error {
	p.BotID = botID
	p.defaultMaxUserMessages = constants.MaxUserMessages
	p.logger = &p.client.Log
	p.poster = &p.client.Post

	p.Channel = builder.NewChannel(p.client)
	p.Store = builder.NewStore(p.client, p.defaultMaxUserMessages)
	p.Scheduler = builder.NewScheduler(p.client, p.Store, p.Channel, p.BotID, clk)
	p.Command = builder.NewCommandHandler(p.client, p.Store, p.Scheduler, p.Channel, p.defaultMaxUserMessages, clk, p.helpText)
	if err := p.Command.Register(); err != nil {
		return err
	}
	go p.Scheduler.Start()
	return nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	return p.Command.Execute(args)
}
