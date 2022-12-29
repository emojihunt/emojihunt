package client

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
)

type DiscordReactionHandler func(*discordgo.Session, *discordgo.MessageReaction, string) error

func (c *Discord) AddReactionHandler(handler *DiscordReactionHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.reactionHandlers = append(c.reactionHandlers, handler)
}
func (c *Discord) reactionAddHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	c.reactionCommon(s, r.MessageReaction, "add ")
}

func (c *Discord) reactionRemoveHandler(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	c.reactionCommon(s, r.MessageReaction, "remove ")
}

func (c *Discord) reactionRemoveAllHandler(s *discordgo.Session, r *discordgo.MessageReactionRemoveAll) {
	c.reactionCommon(s, r.MessageReaction, "remove-all")
}

func (c *Discord) reactionCommon(s *discordgo.Session, r *discordgo.MessageReaction, kind string) {
	for _, handler := range c.reactionHandlers {
		err := (*handler)(s, r, kind)
		if err != nil {
			log.Printf("discord: error handling reaction %s%s on message %q: %s",
				kind, r.Emoji.Name, r.MessageID, spew.Sdump(err))
		}
	}
}
