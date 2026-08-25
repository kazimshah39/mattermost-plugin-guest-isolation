package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestExecuteCommand_GuestInvite(t *testing.T) {
	api := &plugintest.API{}
	p := &Plugin{}
	p.SetAPI(api)

	channelID := "test_channel_123"
	teamID := "test_team_456"

	api.On("GetChannel", channelID).Return(&model.Channel{
		Id:          channelID,
		TeamId:      teamID,
		DisplayName: "Client Acme Project",
		Name:        "client-acme-project",
	}, nil)

	api.On("KVSet", mock.MatchedBy(func(key string) bool {
		return strings.HasPrefix(key, KVInvitePrefix)
	}), mock.Anything).Return(nil)

	siteURL := "http://localhost:8065"
	api.On("GetConfig").Return(&model.Config{
		ServiceSettings: model.ServiceSettings{
			SiteURL: &siteURL,
		},
	})

	resp, appErr := p.ExecuteCommand(nil, &model.CommandArgs{
		Command:   "/guest-invite",
		ChannelId: channelID,
		UserId:    "admin_user_id",
	})

	assert.Nil(t, appErr)
	assert.NotNil(t, resp)
	assert.Contains(t, resp.Text, "Client Invite Link Generated")
	assert.Contains(t, resp.Text, "/plugins/com.custom.guest-isolation/signup?token=")
}

func TestHandleGetSignup_Validation(t *testing.T) {
	api := &plugintest.API{}
	p := &Plugin{}
	p.SetAPI(api)

	// Case 1: Missing token
	req := httptest.NewRequest(http.MethodGet, "/plugins/com.custom.guest-isolation/signup", nil)
	w := httptest.NewRecorder()
	p.ServeHTTP(nil, w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid Invitation")

	// Case 2: Invalid/non-existent token
	api.On("KVGet", KVInvitePrefix+"non_existent_token").Return(nil, nil)
	req2 := httptest.NewRequest(http.MethodGet, "/plugins/com.custom.guest-isolation/signup?token=non_existent_token", nil)
	w2 := httptest.NewRecorder()
	p.ServeHTTP(nil, w2, req2)

	assert.Equal(t, http.StatusNotFound, w2.Code)
	assert.Contains(t, w2.Body.String(), "Invitation Not Found")

	// Case 3: Valid token
	validInvite := InviteRecord{
		Token:     "valid_token_123",
		ChannelID: "ch_123",
		TeamID:    "team_123",
		Used:      false,
	}
	data, _ := json.Marshal(validInvite)
	api.On("KVGet", KVInvitePrefix+"valid_token_123").Return(data, nil)
	api.On("GetChannel", "ch_123").Return(&model.Channel{
		Id:          "ch_123",
		DisplayName: "Client Portal",
	}, nil)

	req3 := httptest.NewRequest(http.MethodGet, "/plugins/com.custom.guest-isolation/signup?token=valid_token_123", nil)
	w3 := httptest.NewRecorder()
	p.ServeHTTP(nil, w3, req3)

	assert.Equal(t, http.StatusOK, w3.Code)
	assert.Contains(t, w3.Body.String(), "Join Project Workspace")
	assert.Contains(t, w3.Body.String(), "Client Portal")
}
