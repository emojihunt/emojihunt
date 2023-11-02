package discord

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
)

type ReactionHandler func(*discordgo.Session, *discordgo.MessageReaction, string) error

func (c *Client) AddReactionHandler(handler *ReactionHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.reactionHandlers = append(c.reactionHandlers, handler)
}
func (c *Client) reactionAddHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	c.reactionCommon(s, r.MessageReaction, "add ")
}

func (c *Client) reactionRemoveHandler(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	c.reactionCommon(s, r.MessageReaction, "remove ")
}

func (c *Client) reactionRemoveAllHandler(s *discordgo.Session, r *discordgo.MessageReactionRemoveAll) {
	c.reactionCommon(s, r.MessageReaction, "remove-all")
}

func (c *Client) reactionCommon(s *discordgo.Session, r *discordgo.MessageReaction, kind string) {
	for _, handler := range c.reactionHandlers {
		err := (*handler)(s, r, kind)
		if err != nil {
			log.Printf("discord: error handling reaction %s%s on message %q: %s",
				kind, r.Emoji.Name, r.MessageID, spew.Sdump(err))
		}
	}
}
