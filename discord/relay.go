package discord

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/state"
	"golang.org/x/xerrors"
)

const (
	relayWebhookName    = "Huntbot Relay"
	ablyRelayEventTitle = "m"
)

// Message Handling

type AblyMessage struct {
	ID        string `json:"id"`
	ChannelID string `json:"ch,omitempty"`
	Author    string `json:"u,omitempty"`
	Timestamp int64  `json:"t"`
	Content   string `json:"msg"` // don't omit (for deletes)
}

func (c *Client) handleMessageCreate(
	ctx context.Context, m *discordgo.MessageCreate,
) error {
	if c.ignoreMessage(m.Message) {
		return nil
	}
	var message = AblyMessage{
		ID:        m.Message.ID,
		ChannelID: m.ChannelID,
		Author:    c.DisplayName(m.Author),
		Timestamp: m.Timestamp.UnixMilli(),
		Content:   m.Message.Content,
	}
	c.state.LiveMessage <- state.LiveMessage{
		Event: ablyRelayEventTitle,
		Data:  message,
	}
	return c.ably.Publish(ctx, ablyRelayEventTitle, message)
}

func (c *Client) handleMessageUpdate(
	ctx context.Context, m *discordgo.MessageUpdate,
) error {
	if c.ignoreMessage(m.Message) {
		return nil
	}
	var message = AblyMessage{
		ID:      m.Message.ID,
		Content: m.Message.Content,
	}
	c.state.LiveMessage <- state.LiveMessage{
		Event: ablyRelayEventTitle,
		Data:  message,
	}
	return c.ably.Publish(ctx, ablyRelayEventTitle, message)
}

func (c *Client) handleMessageDelete(
	ctx context.Context, m *discordgo.MessageDelete,
) error {
	if c.ignoreMessage(m.Message) {
		return nil
	}
	var message = AblyMessage{
		ID: m.Message.ID,
	}
	c.state.LiveMessage <- state.LiveMessage{
		Event: ablyRelayEventTitle,
		Data:  message,
	}
	return c.ably.Publish(ctx, ablyRelayEventTitle, message)
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

func (c *Client) handleGuildMemberAdd(
	ctx context.Context, g *discordgo.GuildMemberAdd,
) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.memberCache[g.User.ID] = g.Member
	log.Printf("discord: member added: %q", g.User.GlobalName)
	return nil
}

func (c *Client) handleGuildMemberUpdate(
	ctx context.Context, g *discordgo.GuildMemberUpdate,
) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.memberCache[g.User.ID] = g.Member
	log.Printf("discord: member updated: %q", g.User.GlobalName)
	return nil
}

func (c *Client) handleGuildMemberRemove(
	ctx context.Context, g *discordgo.GuildMemberRemove,
) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	delete(c.memberCache, g.User.ID)
	log.Printf("discord: member removed: %q", g.User.GlobalName)
	return nil
}

func (c *Client) refreshMemberCache() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	members, err := c.s.GuildMembers(c.Guild.ID, "", 1000)
	if err != nil {
		return err
	}
	c.memberCache = make(map[string]*discordgo.Member)
	for _, member := range members {
		c.memberCache[member.User.ID] = member
	}
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

func (c *Client) refreshWebhookCache() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	webhooks, err := c.s.GuildWebhooks(c.Guild.ID)
	if err != nil {
		return err
	}
	c.webhookCache = make(map[string]*discordgo.Webhook)
	for _, webhook := range webhooks {
		if webhook.ApplicationID != c.Application.ID {
			continue
		}
		c.webhookCache[webhook.ChannelID] = webhook
	}
	return nil
}
