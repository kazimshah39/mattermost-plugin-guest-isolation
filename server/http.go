package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/mattermost/mattermost/server/public/plugin"
)

//go:embed assets/signup.html
var signupHTMLTemplate string

type SignupTemplateData struct {
	Token       string
	ChannelName string
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path

	switch {
	case strings.HasSuffix(path, "/signup") && r.Method == http.MethodGet:
		p.handleGetSignup(w, r)
	case strings.HasSuffix(path, "/api/v1/signup") && r.Method == http.MethodPost:
		p.handlePostSignup(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (p *Plugin) handleGetSignup(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		p.renderErrorPage(w, "Invalid Invitation", "No invitation token was provided. Please request a new invite link from your team administrator.", http.StatusBadRequest)
		return
	}

	data, appErr := p.API.KVGet(KVInvitePrefix + token)
	if appErr != nil || data == nil {
		p.renderErrorPage(w, "Invitation Not Found", "This invitation link is invalid or has expired. Please request a new invite link.", http.StatusNotFound)
		return
	}

	var invite InviteRecord
	if err := json.Unmarshal(data, &invite); err != nil {
		p.renderErrorPage(w, "Server Error", "Unable to process invitation record.", http.StatusInternalServerError)
		return
	}

	if invite.Used {
		p.renderErrorPage(w, "Invitation Already Used", "This invitation link has already been used to create an account. Please log in directly with your email and password.", http.StatusGone)
		return
	}

	channel, appErr := p.API.GetChannel(invite.ChannelID)
	channelName := "Project Channel"
	if appErr == nil && channel != nil {
		channelName = channel.DisplayName
	}

	tmpl, err := template.New("signup").Parse(signupHTMLTemplate)
	if err != nil {
		p.API.LogError("Failed to parse signup HTML template", "err", err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, SignupTemplateData{
		Token:       token,
		ChannelName: channelName,
	})
}

func (p *Plugin) renderErrorPage(w http.ResponseWriter, title, message string, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s - Mattermost</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f4f6f8; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; padding: 20px; }
    .card { background: white; border-radius: 12px; box-shadow: 0 10px 25px rgba(0,0,0,0.05); padding: 36px; max-width: 440px; text-align: center; }
    h1 { color: #dc3545; font-size: 20px; margin-bottom: 12px; }
    p { color: #6c757d; font-size: 14px; line-height: 1.5; margin-bottom: 24px; }
    a { display: inline-block; background: #1e5cb3; color: white; text-decoration: none; padding: 10px 20px; border-radius: 6px; font-weight: 600; font-size: 14px; }
  </style>
</head>
<body>
  <div class="card">
    <h1>⚠️ %s</h1>
    <p>%s</p>
    <a href="/login">Go to Login</a>
  </div>
</body>
</html>`, title, title, message)
}
