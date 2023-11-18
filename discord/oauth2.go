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

func (c *Client) CheckOAuth2Token(ctx context.Context, token string) (*discordgo.Member, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", EndpointOAuth2Session, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if err != nil {
		return nil, xerrors.New("invalid access token")
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, xerrors.New("invalid access token")
	}
	defer res.Body.Close()

	var session OAuth2Session
	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		return nil, xerrors.New("invalid access token")
	} else if session.Application.ID != c.Application.ID {
		return nil, xerrors.Errorf("unexpected application ID: %s", session.Application.ID)
	}
	member, err := c.s.GuildMember(c.Guild.ID, session.User.ID)
	if err != nil {
		return nil, xerrors.Errorf("not a member of guild")
	}
	return member, nil
}
