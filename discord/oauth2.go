package discord

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/xerrors"
)

var EndpointOAuth2Session = discordgo.EndpointOAuth2 + "@me"

type OAuth2Session struct {
	Application discordgo.Application `json:"application"`
	Expires     time.Time             `json:"expires"`
	Scopes      []string              `json:"scopes"`
	User        discordgo.User        `json:"user"`
}

func (c *Client) GetOAuth2Session(ctx context.Context, token string) (*OAuth2Session, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", EndpointOAuth2Session, nil)
	if err != nil {
		return nil, xerrors.Errorf("call to /oauth2/@me failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, xerrors.Errorf("call to /oauth2/@me failed: %w", err)
	}
	defer res.Body.Close()

	var session OAuth2Session
	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return nil, xerrors.Errorf("failed to decode /oauth2/@me response: %w", err)
	}
	return &session, nil
}
