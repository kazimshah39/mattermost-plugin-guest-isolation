package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mattermost/mattermost/server/public/model"
)

type SignupRequest struct {
	Token    string `json:"token"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignupResponse struct {
	Success     bool   `json:"success"`
	RedirectURL string `json:"redirect_url,omitempty"`
	Error       string `json:"error,omitempty"`
}

type ClientMetadata struct {
	UserID    string `json:"user_id"`
	ChannelID string `json:"channel_id"`
	TeamID    string `json:"team_id"`
	CreatedAt int64  `json:"created_at"`
}

var usernameRegex = regexp.MustCompile(`[^a-z0-9\.\-_]`)

func (p *Plugin) handlePostSignup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SignupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(SignupResponse{Success: false, Error: "Invalid JSON request."})
		return
	}

	req.Token = strings.TrimSpace(req.Token)
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Password = strings.TrimSpace(req.Password)

	if req.Token == "" || req.Name == "" || req.Email == "" || len(req.Password) < 8 {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(SignupResponse{
			Success: false,
			Error:   "All fields are required and password must be at least 8 characters.",
		})
		return
	}

	data, appErr := p.API.KVGet(KVInvitePrefix + req.Token)
	if appErr != nil || data == nil {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(SignupResponse{Success: false, Error: "Invalid or expired invitation token."})
		return
	}

	var invite InviteRecord
	if err := json.Unmarshal(data, &invite); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(SignupResponse{Success: false, Error: "Failed to process invitation."})
		return
	}

	if invite.Used {
		w.WriteHeader(http.StatusGone)
		_ = json.NewEncoder(w).Encode(SignupResponse{Success: false, Error: "This invitation link has already been used."})
		return
	}

	nameParts := strings.Fields(req.Name)
	firstName := req.Name
	lastName := ""
	if len(nameParts) > 1 {
		firstName = nameParts[0]
		lastName = strings.Join(nameParts[1:], " ")
	}

	emailPrefix := strings.Split(req.Email, "@")[0]
	sanitizedBase := usernameRegex.ReplaceAllString(strings.ToLower(emailPrefix), "")
	if len(sanitizedBase) > 15 {
		sanitizedBase = sanitizedBase[:15]
	}
	if len(sanitizedBase) < 3 {
		sanitizedBase = "user"
	}
	candidateUsername := fmt.Sprintf("%s.%s", sanitizedBase, uuid.New().String()[:5])

	user := &model.User{
		Username:      candidateUsername,
		Email:         req.Email,
		Password:      req.Password,
		FirstName:     firstName,
		LastName:      lastName,
		EmailVerified: true,
	}

	createdUser, createErr := p.API.CreateUser(user)
	if createErr != nil {
		p.API.LogError("Failed to create client user", "email", req.Email, "err", createErr.Message)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(SignupResponse{
			Success: false,
			Error:   fmt.Sprintf("Could not create account: %s", createErr.Message),
		})
		return
	}

	if _, teamErr := p.API.CreateTeamMember(invite.TeamID, createdUser.Id); teamErr != nil {
		p.API.LogWarn("Could not add client to team directly (may already be member)", "err", teamErr.Message)
	}

	p.isolateClientUser(invite.TeamID, invite.ChannelID, createdUser.Id)

	invite.Used = true
	if updatedData, err := json.Marshal(invite); err == nil {
		_ = p.API.KVSet(KVInvitePrefix+req.Token, updatedData)
	}

	clientMeta := ClientMetadata{
		UserID:    createdUser.Id,
		ChannelID: invite.ChannelID,
		TeamID:    invite.TeamID,
		CreatedAt: time.Now().Unix(),
	}
	if metaData, err := json.Marshal(clientMeta); err == nil {
		_ = p.API.KVSet(KVClientPrefix+createdUser.Id, metaData)
	}

	team, _ := p.API.GetTeam(invite.TeamID)
	channel, _ := p.API.GetChannel(invite.ChannelID)
	redirectURL := "/login"
	if team != nil && channel != nil {
		redirectURL = fmt.Sprintf("/%s/channels/%s", team.Name, channel.Name)
	}

	p.API.LogInfo("Client user successfully registered and isolated",
		"username", createdUser.Username,
		"channel", invite.ChannelID,
		"team", invite.TeamID,
	)

	_ = json.NewEncoder(w).Encode(SignupResponse{
		Success:     true,
		RedirectURL: redirectURL,
	})
}

func (p *Plugin) isolateClientUser(teamID, targetChannelID, userID string) {
	// Remove from default public channels (town-square, off-topic)
	if ts, err := p.API.GetChannelByName(teamID, "town-square", false); err == nil && ts != nil {
		if delErr := p.API.DeleteChannelMember(ts.Id, userID); delErr != nil {
			p.API.LogDebug("Town Square removal note", "msg", delErr.Message)
		}
	}

	if ot, err := p.API.GetChannelByName(teamID, "off-topic", false); err == nil && ot != nil {
		if delErr := p.API.DeleteChannelMember(ot.Id, userID); delErr != nil {
			p.API.LogDebug("Off-Topic removal note", "msg", delErr.Message)
		}
	}

	// Add exclusively to target channel
	if _, addErr := p.API.AddChannelMember(targetChannelID, userID); addErr != nil {
		p.API.LogError("Failed to add client to target channel", "channel", targetChannelID, "user", userID, "err", addErr.Message)
	}
}
