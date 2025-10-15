package sync

import (
	"time"

	"github.com/emojihunt/emojihunt/huntyet"
	"github.com/emojihunt/emojihunt/state"
)

type Meta struct {
	HuntName        string `json:"hunt_name"`
	HuntURL         string `json:"hunt_url"`
	HuntCredentials string `json:"hunt_credentials"`
	LogisticsURL    string `json:"logistics_url"`

	DiscordGuild string `json:"discord_guild"`
	HangingOut   string `json:"hanging_out"`
	NextHunt     string `json:"next_hunt"` // work around Ably bug

	VoiceRooms map[string]string `json:"voice_rooms"`
}

func (c *Client) ComputeMeta(discovery state.DiscoveryConfig) Meta {
	var nextHunt string
	if raw, ok := huntyet.NextHunt(time.Now()); ok {
		nextHunt = raw.Format(time.RFC3339)
	}
	return Meta{
		HuntName:        discovery.HuntName,
		HuntURL:         discovery.HuntURL,
		HuntCredentials: discovery.HuntCredentials,
		LogisticsURL:    discovery.LogisticsURL,

		DiscordGuild: c.discord.Guild.ID,
		HangingOut:   c.discord.HangingOutChannel.ID,
		NextHunt:     nextHunt,

		VoiceRooms: c.discord.ListVoiceChannels(),
	}
}
