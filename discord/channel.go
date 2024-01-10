package discord

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/xerrors"
)

// Event Handlers / Channel Cache Maintenance

func (c *Client) handleChannelCreate(ctx context.Context, r *discordgo.ChannelCreate) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.channelCache[r.ID] = r.Channel
	log.Printf("discord: channel created: %q", r.Channel.Name)
	return nil
}

func (c *Client) handleChannelUpdate(ctx context.Context, r *discordgo.ChannelUpdate) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.channelCache[r.ID] = r.Channel
	log.Printf("discord: channel updated: %q", r.Channel.Name)
	return nil
}

func (c *Client) handleChannelDelete(ctx context.Context, r *discordgo.ChannelDelete) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.channelCache, r.ID)
	log.Printf("discord: channel deleted: %q", r.Channel.Name)
	return nil
}

func (c *Client) refreshChannelCache() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	channels, err := c.s.GuildChannels(c.Guild.ID)
	if err != nil {
		return err
	}
	c.channelCache = make(map[string]*discordgo.Channel)
	for _, channel := range channels {
		c.channelCache[channel.ID] = channel
	}
	return nil
}

// Public API

func (c *Client) GetChannel(id string) (*discordgo.Channel, bool) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	channel, ok := c.channelCache[id]
	return channel, ok
}

func (c *Client) ListChannelsByID() map[string]*discordgo.Channel {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var result = make(map[string]*discordgo.Channel)
	for id, channel := range c.channelCache {
		result[id] = channel
	}
	return result
}

func (c *Client) ListCategoriesByName() map[string]*discordgo.Channel {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var categories = make(map[string]*discordgo.Channel)
	for _, channel := range c.channelCache {
		if channel.Type == discordgo.ChannelTypeGuildCategory {
			categories[channel.Name] = channel
		}
	}
	return categories
}

func (c *Client) ListVoiceChannels() map[string]string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var result = make(map[string]string)
	for id, channel := range c.channelCache {
		if channel.Type == discordgo.ChannelTypeGuildVoice {
			result[id] = channel.Name
		}
	}
	return result
}

func (c *Client) CreateChannel(name string, category string, position int) (*discordgo.Channel, error) {
	if len(name) > 100 {
		name = name[:100]
	}
	ch, err := c.s.GuildChannelCreateComplex(c.Guild.ID, discordgo.GuildChannelCreateData{
		Name:     name,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: category,
		Position: position,
	})
	if err != nil {
		return nil, xerrors.Errorf("CreateChannel: %w", err)
	}
	return ch, nil
}

func (c *Client) CreateCategory(name string, position int) (*discordgo.Channel, error) {
	if len(name) > 100 {
		name = name[:100]
	}
	category, err := c.s.GuildChannelCreateComplex(c.Guild.ID,
		discordgo.GuildChannelCreateData{
			Name:     name,
			Type:     discordgo.ChannelTypeGuildCategory,
			Position: position,
		},
	)
	if err != nil {
		return nil, xerrors.Errorf("CreateCategory: %w", err)
	}
	return category, nil
}

func (c *Client) SetChannelName(chID, name string, position int) error {
	// Note that setting the name, even if it's a no-op, causes the channel's
	// position to be reset.
	if len(name) > 100 {
		name = name[:100]
	}
	_, err := c.s.ChannelEdit(chID, &discordgo.ChannelEdit{
		Name:     name,
		Position: position,
	})
	if err != nil {
		return xerrors.Errorf("SetChannelName.Edit: %w", err)
	}
	return nil
}

func (c *Client) SetChannelCategory(channel string, category string, position int) error {
	_, err := c.s.ChannelEditComplex(channel, &discordgo.ChannelEdit{
		ParentID: category,
		Position: position,
	})
	if err != nil {
		return xerrors.Errorf("error moving channel to category %s: %w", category, err)
	}
	return nil
}

type ChannelOrder struct {
	ID       string
	Position int
}

func (c *Client) SortChannels(order []ChannelOrder) error {
	data := make([]struct {
		ID       string `json:"id"`
		Position int    `json:"position"`
	}, len(order))
	for i, c := range order {
		data[i].ID = c.ID
		data[i].Position = c.Position
	}
	_, err := c.s.RequestWithBucketID("PATCH", discordgo.EndpointGuildChannels(c.Guild.ID),
		data, discordgo.EndpointGuildChannels(c.Guild.ID))
	if err != nil {
		return xerrors.Errorf("SortChannels: %w", err)
	}
	return nil
}
