package client

import (
	"encoding/json"
	"time"

	"github.com/bwmarrin/discordgo"
)

type DiscordScheduledEvent struct {
	ID           string    `json:"id,omitempty"`
	ChannelID    string    `json:"channel_id"`
	Name         string    `json:"name"`
	PrivacyLevel int       `json:"privacy_level"`
	StartTime    time.Time `json:"scheduled_start_time"`
	Description  string    `json:"description,omitempty"`
	EntityType   int       `json:"entity_type"`
	Status       int       `json:"status,omitempty"`
}

func (c *Discord) ListScheduledEvents() (map[string]*DiscordScheduledEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// This endpoint is rate-limited to about one request per 10 seconds (why
	// just this one? we don't know) so we save results to a cache.
	if time.Since(c.scheduledEventsLastUpdate) < 15*time.Second {
		return c.scheduledEventsCache, nil
	}

	bucket := discordgo.EndpointGuild(c.Guild.ID) + "/scheduled-events"
	resp, err := c.s.RequestWithBucketID("GET", bucket, nil, bucket)
	if err != nil {
		return nil, err
	}

	var raw []*DiscordScheduledEvent
	if err := json.Unmarshal(resp, &raw); err != nil {
		return nil, err
	}

	events := make(map[string]*DiscordScheduledEvent)
	for _, event := range raw {
		events[event.ID] = event
	}

	c.scheduledEventsCache = events
	c.scheduledEventsLastUpdate = time.Now()

	return events, nil
}

func (c *Discord) CreateScheduledEvent(event *DiscordScheduledEvent) (*DiscordScheduledEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bucket := discordgo.EndpointGuild(c.Guild.ID) + "/scheduled-events"
	resp, err := c.s.RequestWithBucketID("POST", bucket, event, bucket)
	if err != nil {
		return nil, err
	}

	var created *DiscordScheduledEvent
	if err := json.Unmarshal(resp, &created); err != nil {
		return nil, err
	}

	c.scheduledEventsCache[created.ID] = created

	return created, nil
}

func (c *Discord) UpdateScheduledEvent(event *DiscordScheduledEvent, fields map[string]interface{}) (*DiscordScheduledEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	bucket := discordgo.EndpointGuild(c.Guild.ID) + "/scheduled-events/" + event.ID
	resp, err := c.s.RequestWithBucketID("PATCH", bucket, fields, bucket)
	if err != nil {
		return nil, err
	}

	var modified *DiscordScheduledEvent
	if err := json.Unmarshal(resp, &modified); err != nil {
		return nil, err
	}

	c.scheduledEventsCache[modified.ID] = modified

	return modified, nil
}

func (c *Discord) DeleteScheduledEvent(event *DiscordScheduledEvent) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	bucket := discordgo.EndpointGuild(c.Guild.ID) + "/scheduled-events/" + event.ID
	if _, err := c.s.RequestWithBucketID("DELETE", bucket, nil, bucket); err != nil {
		return err
	}

	delete(c.scheduledEventsCache, event.ID)

	return nil
}
