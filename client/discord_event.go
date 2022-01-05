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

func (c *Discord) ListScheduledEvents() ([]*DiscordScheduledEvent, error) {
	// Rate limit: once every 10 seconds
	bucket := discordgo.EndpointGuild(c.GuildID) + "/scheduled-events"
	resp, err := c.s.RequestWithBucketID("GET", bucket, nil, bucket)
	if err != nil {
		return nil, err
	}

	var events []*DiscordScheduledEvent
	err = json.Unmarshal(resp, &events)
	return events, err
}

func (c *Discord) CreateScheduledEvent(event *DiscordScheduledEvent) (*DiscordScheduledEvent, error) {
	// Rate limit: haven't found it yet
	bucket := discordgo.EndpointGuild(c.GuildID) + "/scheduled-events"
	resp, err := c.s.RequestWithBucketID("POST", bucket, event, bucket)
	if err != nil {
		return nil, err
	}

	var created *DiscordScheduledEvent
	err = json.Unmarshal(resp, &created)
	return created, err
}

func (c *Discord) UpdateScheduledEvent(event *DiscordScheduledEvent, fields map[string]interface{}) (*DiscordScheduledEvent, error) {
	bucket := discordgo.EndpointGuild(c.GuildID) + "/scheduled-events/" + event.ID
	resp, err := c.s.RequestWithBucketID("PATCH", bucket, fields, bucket)
	if err != nil {
		return nil, err
	}

	var modified *DiscordScheduledEvent
	err = json.Unmarshal(resp, &modified)
	return modified, err
}

func (c *Discord) DeleteScheduledEvent(event *DiscordScheduledEvent) error {
	bucket := discordgo.EndpointGuild(c.GuildID) + "/scheduled-events/" + event.ID
	_, err := c.s.RequestWithBucketID("DELETE", bucket, nil, bucket)
	return err
}
