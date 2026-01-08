package discord

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/state"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/xerrors"
)

const (
	relayWebhookName = "Huntbot Relay"
)

var (
	messagesRelayed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "discord_relay",
		Help: "The total number of messages relayed from Discord",
	})
)

type UsersMessage struct {
	Users   map[string][2]string `json:"users"`
	Delete  []string             `json:"delete,omitempty"`
	Replace bool                 `json:"replace,omitempty"`
}

func (m *UsersMessage) EventType() state.EventType {
	return state.EventTypeUsers
}

// Message Handling

type AblyMessage struct {
	ID        string `json:"id"`
	ChannelID string `json:"ch,omitempty"`
	Author    string `json:"u,omitempty"`
	Timestamp int64  `json:"t"`
	Content   string `json:"msg"` // don't omit (for deletes)
}

func (m AblyMessage) EventType() state.EventType {
	return state.EventTypeDiscord
}

func (c *Client) handleMessageCreate(
	ctx context.Context, m *discordgo.MessageCreate,
) error {
	if c.ignoreMessage(m.Message) {
		return nil
	}
	messagesRelayed.Inc()
	var message = AblyMessage{
		ID:        m.Message.ID,
		ChannelID: m.ChannelID,
		Author:    c.DisplayName(m.Author),
		Timestamp: m.Timestamp.UnixMilli(),
		Content:   m.Message.Content,
	}
	c.state.LiveMessage <- message
	return c.ably.Publish(ctx, state.EventTypeDiscord, message)
}

func (c *Client) handleMessageUpdate(
	ctx context.Context, m *discordgo.MessageUpdate,
) error {
	if c.ignoreMessage(m.Message) {
		return nil
	}
	messagesRelayed.Inc()
	var message = AblyMessage{
		ID:      m.Message.ID,
		Content: m.Message.Content,
	}
	c.state.LiveMessage <- message
	return c.ably.Publish(ctx, state.EventTypeDiscord, message)
}

func (c *Client) handleMessageDelete(
	ctx context.Context, m *discordgo.MessageDelete,
) error {
	if c.ignoreMessage(m.Message) {
		return nil
	}
	messagesRelayed.Inc()
	var message = AblyMessage{
		ID: m.Message.ID,
	}
	c.state.LiveMessage <- message
	return c.ably.Publish(ctx, state.EventTypeDiscord, message)
}

func (c *Client) ignoreMessage(m *discordgo.Message) bool {
	ch, ok := c.GetChannel(m.ChannelID)
	if !ok || ch.ParentID == c.TeamCategoryID {
		return true // skip messages to #hanging-out, etc. (load management)
	} else if m.Thread != nil {
		return true // skip messages in threads
	} else if ch.Type != discordgo.ChannelTypeGuildText {
		return true // skip messages in voice channels, etc.
	} else if m.Author != nil && m.Author.ID == c.Application.ID {
		return true // skip messages from huntbot
	}
	return false
}

// Guild Member Handling

func (c *Client) DisplayName(u *discordgo.User) string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	member, ok := c.memberCache[u.ID]
	if ok {
		return member.DisplayName()
	} else if u.GlobalName != "" {
		return u.GlobalName
	} else {
		return u.Username // for webhooks
	}
}

func (c *Client) DisplayAvatar(u *discordgo.User) string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	member, ok := c.memberCache[u.ID]
	if ok {
		return member.AvatarURL("")
	}
	return u.AvatarURL("")
}

func OptimizedAvatarURL(m *discordgo.Member) string {
	// Gets the user's avatar, preferring WebP where available.
	if m.Avatar != "" {
		return fmt.Sprintf("guilds/%s/users/%s/avatars/%s.webp", m.GuildID, m.User.ID, m.Avatar)
	} else if m.User.Avatar != "" {
		return fmt.Sprintf("avatars/%s/%s.webp", m.User.ID, m.User.Avatar)
	} else {
		return fmt.Sprintf("embed/avatars/%d.png", m.User.DefaultAvatarIndex()) // png-only
	}
}

func (c *Client) UserList() map[string][2]string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var result = make(map[string][2]string)
	for id, member := range c.memberCache {
		result[id] = [2]string{member.DisplayName(), OptimizedAvatarURL(member)}
	}
	return result
}

func (c *Client) handleGuildMemberAdd(
	ctx context.Context, g *discordgo.GuildMemberAdd,
) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.memberCache[g.User.ID] = g.Member
	c.state.LiveMessage <- &UsersMessage{
		Users: map[string][2]string{
			g.User.ID: {g.Member.DisplayName(), OptimizedAvatarURL(g.Member)},
		},
	}
	log.Printf("discord: member added: %q", g.User.GlobalName)
	return nil
}

func (c *Client) handleGuildMemberUpdate(
	ctx context.Context, g *discordgo.GuildMemberUpdate,
) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.memberCache[g.User.ID] = g.Member
	c.state.LiveMessage <- &UsersMessage{
		Users: map[string][2]string{
			g.User.ID: {g.Member.DisplayName(), OptimizedAvatarURL(g.Member)},
		},
	}
	log.Printf("discord: member updated: %q", g.User.GlobalName)
	return nil
}

func (c *Client) handleGuildMemberRemove(
	ctx context.Context, g *discordgo.GuildMemberRemove,
) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.memberCache, g.User.ID)
	c.state.LiveMessage <- &UsersMessage{
		Delete: []string{g.User.ID},
	}
	log.Printf("discord: member removed: %q", g.User.GlobalName)
	return nil
}

// Webhook Handling

func (c *Client) RelayMessage(chID, userID, msg string) error {
	webhook, err := c.getRelayWebhook(chID)
	if err != nil {
		return err
	}
	_, err = c.s.WebhookExecute(webhook.ID, webhook.Token, true, &discordgo.WebhookParams{
		Content:   msg,
		Username:  c.DisplayName(&discordgo.User{ID: userID}),
		AvatarURL: c.DisplayAvatar(&discordgo.User{ID: userID}),
	})
	if err != nil {
		return xerrors.Errorf("WebhookExecute: %w", err)
	}
	return nil
}

func (c *Client) getRelayWebhook(chID string) (*discordgo.Webhook, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	webhook, ok := c.webhookCache[chID]
	if ok {
		return webhook, nil
	}
	webhook, err := c.s.WebhookCreate(chID, relayWebhookName, "")
	if err != nil {
		return nil, xerrors.Errorf("WebhookCreate: %w", err)
	}
	c.webhookCache[chID] = webhook
	return webhook, nil
}
