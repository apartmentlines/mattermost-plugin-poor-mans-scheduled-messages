package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

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
	configuration *configuration
	client        *pluginapi.Client
	Scheduler     *scheduler.Scheduler
	Store         store.Store
	Command       *command.Handler
	helpText      string
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
	if err := p.loadHelpText(); err != nil {
		p.API.LogError("Plugin activation failed: could not load help text.", "error", err.Error())
		return err
	}
	kvStore := store.NewKVStore(p.client)
	sched := scheduler.New(p.client, kvStore)
	p.Scheduler = sched
	p.Store = kvStore
	p.Command = command.NewHandler(p.client, kvStore, sched, p.helpText)
	if err := p.Command.Register(); err != nil {
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
