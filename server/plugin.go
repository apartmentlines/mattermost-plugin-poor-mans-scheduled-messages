package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/channel"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/clock"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/command"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/scheduler"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/store"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
	"github.com/mattermost/mattermost/server/public/pluginapi"
)

type Plugin struct {
	plugin.MattermostPlugin
	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex
	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration   *configuration
	client          *pluginapi.Client
	BotID           string
	Scheduler       *scheduler.Scheduler
	Store           store.Store
	Channel         *channel.Channel
	Command         *command.Handler
	maxUserMessages int
	helpText        string
	logger          ports.Logger
	poster          ports.PostService
}

func (p *Plugin) loadHelpText() error {
	bundlePath, err := p.API.GetBundlePath()
	if err != nil {
		return fmt.Errorf("failed to get bundle path: %w", err)
	}
	helpFilePath := filepath.Join(bundlePath, "assets", "help.md")
	helpBytes, err := os.ReadFile(helpFilePath)
	if err != nil {
		return fmt.Errorf("failed to read help file %s: %w", helpFilePath, err)
	}
	p.helpText = string(helpBytes)
	return nil
}

func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)
	if helpErr := p.loadHelpText(); helpErr != nil {
		p.API.LogError("Plugin activation failed: could not load help text.", "error", helpErr.Error())
		return helpErr
	}
	botID, botErr := EnsureBot(&p.client.Bot)
	if botErr != nil {
		p.API.LogError("Plugin activation failed: could not ensure bot.", "error", botErr.Error())
		return botErr
	}
	if initErr := p.initialize(botID); initErr != nil {
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

func (p *Plugin) initialize(botID string) error {
	p.BotID = botID
	p.maxUserMessages = 1000
	realClk := clock.NewReal()
	p.logger = &p.client.Log
	p.poster = &p.client.Post

	p.Channel = channel.New(
		p.logger,
		&p.client.Channel,
		&p.client.Team,
		&p.client.User,
	)

	kvStore := store.NewKVStore(
		&p.client.Log,
		&p.client.KV,
		p.maxUserMessages,
	)
	p.Store = kvStore

	sched := scheduler.New(
		&p.client.Log,
		&p.client.Post,
		kvStore,
		p.Channel,
		p.BotID,
		realClk,
	)
	p.Scheduler = sched

	p.Command = command.NewHandler(
		&p.client.Log,
		&p.client.SlashCommand,
		&p.client.User,
		kvStore,
		sched,
		p.Channel,
		p.maxUserMessages,
		realClk,
		p.helpText,
	)
	if err := p.Command.Register(); err != nil {
		return err
	}
	go p.Scheduler.Start()
	return nil
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	return p.Command.Execute(args)
}
