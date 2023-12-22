package discord

import (
	"context"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/xerrors"
)

// Voice Channel Cache

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
	log.Printf("voice room added: %q", r.Channel.Name)
	return nil
}
func (c *Client) handleChannelUpdate(ctx context.Context, r *discordgo.ChannelUpdate) error {
	if r.Channel.Type != discordgo.ChannelTypeGuildVoice {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.voiceRooms[r.ID] = r.Channel
	log.Printf("voice room renamed: %q", r.Channel.Name)
	return nil
}

func (c *Client) handleChannelDelete(ctx context.Context, r *discordgo.ChannelDelete) error {
	if r.Channel.Type != discordgo.ChannelTypeGuildVoice {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.voiceRooms, r.ID)
	log.Printf("voice room removed: %q", r.Channel.Name)
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

// Scheduled Events API

func (c *Client) GetScheduledEvent(id string) (*discordgo.GuildScheduledEvent, error) {
	return c.s.GuildScheduledEvent(c.Guild.ID, id, false)
}

func (c *Client) ListScheduledEvents() (map[string]*discordgo.GuildScheduledEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// This endpoint is rate-limited to about one request per 10 seconds (why
	// just this one? we don't know) so we save results to a cache.
	if time.Since(c.scheduledEventsLastUpdate) < 15*time.Second {
		return c.scheduledEventsCache, nil
	}

	raw, err := c.s.GuildScheduledEvents(c.Guild.ID, false)
	if err != nil {
		return nil, xerrors.Errorf("GuildScheduledEvents: %w", err)
	}

	events := make(map[string]*discordgo.GuildScheduledEvent)
	for _, event := range raw {
		events[event.ID] = event
	}
	c.scheduledEventsCache = events
	c.scheduledEventsLastUpdate = time.Now()
	return events, nil
}

func (c *Client) CreateScheduledEvent(
	params *discordgo.GuildScheduledEventParams,
) (*discordgo.GuildScheduledEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	event, err := c.s.GuildScheduledEventCreate(c.Guild.ID, params)
	if err != nil {
		return nil, xerrors.Errorf("GuildScheduledEventCreate: %w", err)
	}
	c.scheduledEventsCache[event.ID] = event
	return event, nil
}

func (c *Client) UpdateScheduledEvent(
	event *discordgo.GuildScheduledEvent, params *discordgo.GuildScheduledEventParams,
) (*discordgo.GuildScheduledEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var err error
	event, err = c.s.GuildScheduledEventEdit(c.Guild.ID, event.ID, params)
	if err != nil {
		return nil, xerrors.Errorf("GuildScheduledEventEdit: %w", err)
	}
	c.scheduledEventsCache[event.ID] = event
	return event, nil
}

func (c *Client) DeleteScheduledEvent(event *discordgo.GuildScheduledEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.s.GuildScheduledEventDelete(c.Guild.ID, event.ID)
	if err != nil {
		return xerrors.Errorf("GuildScheduledEventDelete: %w", err)
	}
	delete(c.scheduledEventsCache, event.ID)
	return nil
}
