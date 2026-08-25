package main

import (
	"encoding/json"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin"
)

func (p *Plugin) UserHasBeenCreated(c *plugin.Context, user *model.User) {
	data, err := p.API.KVGet(KVClientPrefix + user.Id)
	if err != nil || data == nil {
		return
	}

	var meta ClientMetadata
	if jsonErr := json.Unmarshal(data, &meta); jsonErr != nil {
		return
	}

	p.isolateClientUser(meta.TeamID, meta.ChannelID, user.Id)
}

func (p *Plugin) ChannelHasBeenCreated(c *plugin.Context, channel *model.Channel) {
	if channel.Type != model.ChannelTypeDirect && channel.Type != model.ChannelTypeGroup {
		return
	}

	members, err := p.API.GetChannelMembers(channel.Id, 0, 100)
	if err != nil {
		return
	}

	for _, member := range members {
		clientData, kvErr := p.API.KVGet(KVClientPrefix + member.UserId)
		if kvErr == nil && clientData != nil {
			p.API.LogWarn("Blocking Direct/Group message channel creation for client user",
				"channel_id", channel.Id,
				"user_id", member.UserId,
			)

			// Delete client membership from unauthorized direct message channel
			_ = p.API.DeleteChannelMember(channel.Id, member.UserId)
			break
		}
	}
}

func (p *Plugin) MessageWillBePosted(c *plugin.Context, post *model.Post) (*model.Post, string) {
	if post == nil || post.UserId == "" {
		return post, ""
	}

	clientData, err := p.API.KVGet(KVClientPrefix + post.UserId)
	if err != nil || clientData == nil {
		return post, ""
	}

	var meta ClientMetadata
	if jsonErr := json.Unmarshal(clientData, &meta); jsonErr != nil {
		return post, ""
	}

	// If client tries to post to a channel other than their assigned channel, reject post
	if post.ChannelId != meta.ChannelID {
		p.API.LogWarn("Client user attempted to post in unassigned channel",
			"user_id", post.UserId,
			"channel_id", post.ChannelId,
			"allowed_channel_id", meta.ChannelID,
		)
		return nil, "You are only permitted to post messages inside your assigned project channel."
	}

	return post, ""
}

func (p *Plugin) FilterPost(c *plugin.Context, post *model.Post) (*model.Post, string) {
	return p.MessageWillBePosted(c, post)
}
