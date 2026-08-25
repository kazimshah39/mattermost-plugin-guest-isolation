package main

import (
	"sync"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

const (
	PluginID        = "com.custom.guest-isolation"
	CommandTrigger  = "guest-invite"
	KVInvitePrefix  = "invite:"
	KVClientPrefix  = "client:"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex
}

func (p *Plugin) OnActivate() error {
	p.API.LogInfo("Guest Auto-Isolation Plugin activating...")

	if err := p.API.RegisterCommand(&model.Command{
		Trigger:          CommandTrigger,
		AutoComplete:     true,
		AutoCompleteDesc: "Generate a 1-click isolated client signup link for this channel",
		AutoCompleteHint: "[optional note]",
		DisplayName:      "Guest Invite",
		Description:      "Generates a client invitation link that isolates the user to this specific channel.",
	}); err != nil {
		p.API.LogError("Failed to register command", "err", err.Error())
		return err
	}

	p.API.LogInfo("Guest Auto-Isolation Plugin activated successfully")
	return nil
}

func (p *Plugin) OnDeactivate() error {
	p.API.LogInfo("Guest Auto-Isolation Plugin deactivated")
	return nil
}
