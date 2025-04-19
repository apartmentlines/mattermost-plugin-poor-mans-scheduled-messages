package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/ports"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/internal/testutil"
	"github.com/apartmentlines/mattermost-plugin-poor-mans-scheduled-messages/server/constants"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/mattermost/mattermost/server/public/pluginapi"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func pluginTestAPI() *plugintest.API {
	api := &plugintest.API{}
	api.On("LogInfo", mock.Anything, mock.Anything).Maybe()
	api.On("LogError", mock.Anything, mock.Anything, mock.Anything).Maybe()
	return api
}

func TestLoadHelpTextBypass(t *testing.T) {
	p := &Plugin{}
	p.API = pluginTestAPI()
	got, err := p.loadHelpText("inline")
	require.NoError(t, err)
	require.Equal(t, "inline", got)
}

func TestLoadHelpTextFromFile(t *testing.T) {
	tmp := t.TempDir()
	assets := filepath.Join(tmp, "assets")
	_ = os.MkdirAll(assets, 0700)
	require.NoError(t, os.WriteFile(filepath.Join(assets, "help.md"), []byte("file"), 0600))

	api := pluginTestAPI()
	api.On("GetBundlePath").Return(tmp, nil)

	p := &Plugin{}
	p.API = api
	got, err := p.loadHelpText("")
	require.NoError(t, err)
	require.Equal(t, "file", got)
}

func TestLoadHelpTextBundleError(t *testing.T) {
	api := pluginTestAPI()
	api.On("GetBundlePath").Return("", errors.New("fail"))

	p := &Plugin{}
	p.API = api
	_, err := p.loadHelpText("")
	require.Error(t, err)
}

func TestLoadHelpTextReadError(t *testing.T) {
	tmp := t.TempDir()

	api := pluginTestAPI()
	api.On("GetBundlePath").Return(tmp, nil)

	p := &Plugin{}
	p.API = api
	_, err := p.loadHelpText("")
	require.Error(t, err)
}

func TestOnActivateWithSuccess(t *testing.T) {
	api := pluginTestAPI()
	api.On("RegisterCommand", mock.Anything).Return(nil)

	clk := func() ports.Clock { return testutil.FakeClock{NowTime: time.Now()} }

	pl := &Plugin{}
	pl.API = api
	pl.Driver = &plugintest.Driver{}

	stubOK := func(ports.BotService, ports.BotProfileImageService) (string, error) {
		return "bot-id", nil
	}

	err := pl.OnActivateWith(pluginapi.NewClient,
		clk,
		nil,
		stubOK,
		"help")
	require.NoError(t, err)

	require.Equal(t, "help", pl.helpText)

	// verify initialize populated key fields
	require.NotNil(t, pl.Channel)
	require.NotNil(t, pl.Store)
	require.NotNil(t, pl.Scheduler)
	require.NotNil(t, pl.Command)
	require.Equal(t, "bot-id", pl.BotID)
	require.Equal(t, constants.MaxUserMessages, pl.defaultMaxUserMessages)
	require.Equal(t, &pl.client.Log, pl.logger)
	require.Equal(t, &pl.client.Post, pl.poster)
	require.NoError(t, pl.OnDeactivate())
}

func TestOnActivateWithBotError(t *testing.T) {
	api := pluginTestAPI()

	pl := &Plugin{}
	pl.API = api
	pl.Driver = &plugintest.Driver{}

	stubErr := func(ports.BotService, ports.BotProfileImageService) (string, error) {
		return "", errors.New("bot err")
	}

	err := pl.OnActivateWith(pluginapi.NewClient,
		func() ports.Clock { return testutil.FakeClock{NowTime: time.Now()} },
		nil,
		stubErr,
		"help")
	require.Error(t, err)
}

func TestOnActivateHelpTextError(t *testing.T) {
	api := pluginTestAPI()
	api.On("GetBundlePath").Return("", errors.New("bundle path err"))

	p := &Plugin{}
	p.API = api
	p.Driver = &plugintest.Driver{}

	stubOK := func(ports.BotService, ports.BotProfileImageService) (string, error) {
		return "bot-id", nil
	}

	err := p.OnActivateWith(pluginapi.NewClient,
		func() ports.Clock { return testutil.FakeClock{NowTime: time.Now()} },
		nil,
		stubOK,
		"")
	require.Error(t, err)
}

func TestOnActivateWithRegisterError(t *testing.T) {
	api := pluginTestAPI()
	api.On("RegisterCommand", mock.Anything).Return(errors.New("register-fail"))

	pl := &Plugin{}
	pl.API = api
	pl.Driver = &plugintest.Driver{}

	stubOK := func(ports.BotService, ports.BotProfileImageService) (string, error) {
		return "bot-id", nil
	}

	err := pl.OnActivateWith(pluginapi.NewClient,
		func() ports.Clock { return testutil.FakeClock{NowTime: time.Now()} },
		nil,
		stubOK,
		"help")
	require.Error(t, err)
}
