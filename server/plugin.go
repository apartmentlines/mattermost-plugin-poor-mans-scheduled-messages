package main

import (
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
}

func (p *Plugin) OnActivate() error {
	p.client = pluginapi.NewClient(p.API, p.Driver)
	kvStore := store.NewKVStore(p.client)
	sched := scheduler.New(p.client, kvStore)
	p.Scheduler = sched
	p.Store = kvStore
	p.Command = command.NewHandler(p.client, kvStore, sched)
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
