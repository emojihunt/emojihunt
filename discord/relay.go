package discord

import (
	"context"
	"log"

	"github.com/bwmarrin/discordgo"
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
	if c.ignoreMessage(m.Message, false) {
		return nil
	}
	var message = AblyMessage{
		ID:        m.Message.ID,
		ChannelID: m.ChannelID,
		Author:    c.DisplayName(m.Author),
		Timestamp: m.Timestamp.UnixMilli(),
		Content:   m.Message.Content,
	}
	return c.ably.Publish(ctx, "m", message)
}

func (c *Client) handleMessageUpdate(
	ctx context.Context, m *discordgo.MessageUpdate,
) error {
	if c.ignoreMessage(m.Message, false) {
		return nil
	}
	var message = AblyMessage{
		ID:      m.Message.ID,
		Content: m.Message.Content,
	}
	return c.ably.Publish(ctx, "m", message)
}

func (c *Client) handleMessageDelete(
	ctx context.Context, m *discordgo.MessageDelete,
) error {
	if c.ignoreMessage(m.Message, true) {
		return nil
	}
	var message = AblyMessage{
		ID: m.Message.ID,
	}
	return c.ably.Publish(ctx, "m", message)
}

func (c *Client) ignoreMessage(m *discordgo.Message, delete bool) bool {
	ch, ok := c.GetChannel(m.ChannelID)
	if !ok || ch.ParentID == c.TeamCategoryID {
		return true // skip messages to #hanging-out, etc. (load management)
	} else if !delete && (m.Author == nil || m.Author.Bot) {
		return true // skip bot messages (avoid loops)
	} else if m.Thread != nil {
		return true // skip messages in threads
	} else if ch.Type != discordgo.ChannelTypeGuildText {
		return true // skip messages in voice channels, etc.
	}
	return false
}

func (c *Client) DisplayName(u *discordgo.User) string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	member, ok := c.memberCache[u.ID]
	if ok && member.Nick != "" {
		return member.Nick
	} else if u.GlobalName != "" {
		return u.GlobalName
	} else {
		return u.Username
	}
}

// Guild Member Handling

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
