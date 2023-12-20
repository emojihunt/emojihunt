package discord

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

func (c *Client) ListVoiceChannels() map[string]string {
	c.mu.Lock()
	defer c.mu.Unlock()
	var result = make(map[string]string)
	for id, channel := range c.voiceRooms {
		result[id] = channel.Name
	}
	return result
}

func (c *Client) handleChannelCreate(ctx context.Context, r *discordgo.ChannelCreate) error {
	if r.Channel.Type != discordgo.ChannelTypeGuildVoice {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.voiceRooms[r.ID] = r.Channel
	return nil
}
func (c *Client) handleChannelUpdate(ctx context.Context, r *discordgo.ChannelUpdate) error {
	if r.Channel.Type != discordgo.ChannelTypeGuildVoice {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.voiceRooms[r.ID] = r.Channel
	return nil
}

func (c *Client) handleChannelDelete(ctx context.Context, r *discordgo.ChannelDelete) error {
	if r.Channel.Type != discordgo.ChannelTypeGuildVoice {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.voiceRooms, r.ID)
	return nil
}

func (c *Client) refreshVoiceChannels() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	channels, err := c.s.GuildChannels(c.Guild.ID)
	if err != nil {
		return err
	}
	c.voiceRooms = make(map[string]*discordgo.Channel)
	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildVoice {
			c.voiceRooms[channel.ID] = channel
		}
	}
	return nil
}
