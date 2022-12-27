package client

import (
	"encoding/json"
	"time"

	"github.com/bwmarrin/discordgo"
)

func (c *Discord) GetScheduledEvent(id string) (*discordgo.GuildScheduledEvent, error) {
	return c.s.GuildScheduledEvent(c.Guild.ID, id, false)
}

func (c *Discord) ListScheduledEvents() (map[string]*discordgo.GuildScheduledEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// This endpoint is rate-limited to about one request per 10 seconds (why
	// just this one? we don't know) so we save results to a cache.
	if time.Since(c.scheduledEventsLastUpdate) < 15*time.Second {
		return c.scheduledEventsCache, nil
	}

	raw, err := c.s.GuildScheduledEvents(c.Guild.ID, false)
	if err != nil {
		return nil, err
	}

	events := make(map[string]*discordgo.GuildScheduledEvent)
	for _, event := range raw {
		events[event.ID] = event
	}

	c.scheduledEventsCache = events
	c.scheduledEventsLastUpdate = time.Now()

	return events, nil
}

func (c *Discord) CreateScheduledEvent(params *discordgo.GuildScheduledEventParams) (*discordgo.GuildScheduledEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	event, err := c.s.GuildScheduledEventCreate(c.Guild.ID, params)
	if err != nil {
		return nil, err
	}

	c.scheduledEventsCache[event.ID] = event

	return event, nil
}

func (c *Discord) UpdateScheduledEvent(event *discordgo.GuildScheduledEvent, fields map[string]interface{}) (*discordgo.GuildScheduledEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var err error
	body, err := c.s.RequestWithBucketID("PATCH", discordgo.EndpointGuildScheduledEvent(c.Guild.ID, event.ID), fields, discordgo.EndpointGuildScheduledEvent(c.Guild.ID, event.ID))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &event)
	if err != nil {
		return nil, err
	}

	c.scheduledEventsCache[event.ID] = event

	return event, nil
}

func (c *Discord) DeleteScheduledEvent(event *discordgo.GuildScheduledEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.s.GuildScheduledEventDelete(c.Guild.ID, event.ID)
	if err != nil {
		return err
	}

	delete(c.scheduledEventsCache, event.ID)

	return nil
}
