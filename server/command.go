package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

type InviteRecord struct {
	Token     string `json:"token"`
	ChannelID string `json:"channel_id"`
	TeamID    string `json:"team_id"`
	CreatorID string `json:"creator_id"`
	CreatedAt int64  `json:"created_at"`
	Used      bool   `json:"used"`
}

func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	if !strings.HasPrefix(args.Command, "/"+CommandTrigger) {
		return &model.CommandResponse{}, nil
	}

	channel, appErr := p.API.GetChannel(args.ChannelId)
	if appErr != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("❌ Error retrieving channel: %s", appErr.Message),
		}, nil
	}

	token := uuid.New().String()
	record := InviteRecord{
		Token:     token,
		ChannelID: channel.Id,
		TeamID:    channel.TeamId,
		CreatorID: args.UserId,
		CreatedAt: time.Now().Unix(),
		Used:      false,
	}

	data, err := json.Marshal(record)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         "❌ Failed to encode invite record.",
		}, nil
	}

	if setErr := p.API.KVSet(KVInvitePrefix+token, data); setErr != nil {
		return &model.CommandResponse{
			ResponseType: model.CommandResponseTypeEphemeral,
			Text:         fmt.Sprintf("❌ Failed to save invite in KV store: %s", setErr.Message),
		}, nil
	}

	siteURL := ""
	config := p.API.GetConfig()
	if config != nil && config.ServiceSettings.SiteURL != nil && *config.ServiceSettings.SiteURL != "" {
		siteURL = strings.TrimRight(*config.ServiceSettings.SiteURL, "/")
	}
	if siteURL == "" {
		siteURL = "http://localhost:8065"
	}

	signupURL := fmt.Sprintf("%s/plugins/%s/signup?token=%s", siteURL, PluginID, token)

	text := fmt.Sprintf(`### 🔗 Client Invite Link Generated
**Channel:** **#%s**
**Invite URL:** [%s](%s)

Send this link to your client. When they complete the 1-step signup:
- They will be added **exclusively to this channel**.
- Default public channels (*Town Square*, *Off-Topic*) will be automatically excluded.
- They will be able to log in from the **Web, Desktop, and Mobile Apps (iOS & Android)**.`,
		channel.DisplayName, signupURL, signupURL)

	return &model.CommandResponse{
		ResponseType: model.CommandResponseTypeEphemeral,
		Text:         text,
	}, nil
}
