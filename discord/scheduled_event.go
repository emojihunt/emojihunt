package discord

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/xerrors"
)

func (c *Client) GetScheduledEvent(id string) (*discordgo.GuildScheduledEvent, error) {
	return c.s.GuildScheduledEvent(c.Guild.ID, id, false)
}

func (c *Client) ListScheduledEvents() (map[string]*discordgo.GuildScheduledEvent, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

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
	c.mutex.Lock()
	defer c.mutex.Unlock()

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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var err error
	event, err = c.s.GuildScheduledEventEdit(c.Guild.ID, event.ID, params)
	if err != nil {
		return nil, xerrors.Errorf("GuildScheduledEventEdit: %w", err)
	}
	c.scheduledEventsCache[event.ID] = event
	return event, nil
}

func (c *Client) DeleteScheduledEvent(event *discordgo.GuildScheduledEvent) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	err := c.s.GuildScheduledEventDelete(c.Guild.ID, event.ID)
	if err != nil {
		return xerrors.Errorf("GuildScheduledEventDelete: %w", err)
	}
	delete(c.scheduledEventsCache, event.ID)
	return nil
}
