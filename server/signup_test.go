package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mattermost/mattermost/server/public/model"
	"github.com/mattermost/mattermost/server/public/plugin/plugintest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestHandlePostSignup_FullFlow(t *testing.T) {
	api := &plugintest.API{}
	p := &Plugin{}
	p.SetAPI(api)

	token := "token_test_abc"
	channelID := "target_channel_789"
	teamID := "team_456"

	invite := InviteRecord{
		Token:     token,
		ChannelID: channelID,
		TeamID:    teamID,
		Used:      false,
	}
	inviteBytes, _ := json.Marshal(invite)

	// Mock KVGet for token
	api.On("KVGet", KVInvitePrefix+token).Return(inviteBytes, nil)

	// Mock CreateUser
	createdUserID := "created_client_user_id"
	api.On("CreateUser", mock.MatchedBy(func(u *model.User) bool {
		return u.Email == "client@test.com" && u.FirstName == "Client"
	})).Return(&model.User{
		Id:        createdUserID,
		Username:  "client.test",
		Email:     "client@test.com",
		FirstName: "Client",
		LastName:  "User",
	}, nil)

	// Mock CreateTeamMember
	api.On("CreateTeamMember", teamID, createdUserID).Return(&model.TeamMember{}, nil)

	// Mock GetChannelByName for town-square and off-topic
	api.On("GetChannelByName", teamID, "town-square", false).Return(&model.Channel{Id: "town_square_id"}, nil)
	api.On("GetChannelByName", teamID, "off-topic", false).Return(&model.Channel{Id: "off_topic_id"}, nil)

	// Mock DeleteChannelMember
	api.On("DeleteChannelMember", "town_square_id", createdUserID).Return(nil)
	api.On("DeleteChannelMember", "off_topic_id", createdUserID).Return(nil)

	// Mock AddChannelMember for target channel
	api.On("AddChannelMember", channelID, createdUserID).Return(&model.ChannelMember{}, nil)

	// Mock KVSet for token update and client metadata
	api.On("KVSet", KVInvitePrefix+token, mock.Anything).Return(nil)
	api.On("KVSet", KVClientPrefix+createdUserID, mock.Anything).Return(nil)

	// Mock GetTeam and GetChannel for redirect URL
	api.On("GetTeam", teamID).Return(&model.Team{Name: "devteam"}, nil)
	api.On("GetChannel", channelID).Return(&model.Channel{Name: "client-project"}, nil)

	api.On("LogInfo", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	body, _ := json.Marshal(SignupRequest{
		Token:    token,
		Name:     "Client User",
		Email:    "client@test.com",
		Password: "Password123!",
	})

	req := httptest.NewRequest(http.MethodPost, "/plugins/com.custom.guest-isolation/api/v1/signup", bytes.NewReader(body))
	w := httptest.NewRecorder()

	p.ServeHTTP(nil, w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp SignupResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Nil(t, err)
	assert.True(t, resp.Success)
	assert.Equal(t, "/devteam/channels/client-project", resp.RedirectURL)
}
