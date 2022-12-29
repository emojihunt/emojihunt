package client

import (
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
)

type DiscordReactionHandler func(*discordgo.Session, *discordgo.MessageReaction, *discordgo.Message) error

func (c *Discord) reactionAddHandler(s *discordgo.Session, r *discordgo.MessageReactionAdd) {
	c.reactionCommon(s, "add", r.MessageReaction)
}

func (c *Discord) reactionRemoveHandler(s *discordgo.Session, r *discordgo.MessageReactionRemove) {
	c.reactionCommon(s, "remove", r.MessageReaction)
}

func (c *Discord) reactionRemoveAllHandler(s *discordgo.Session, r *discordgo.MessageReactionRemoveAll) {
	c.reactionCommon(s, "remove all", r.MessageReaction)
}

func (c *Discord) reactionCommon(s *discordgo.Session, kind string, r *discordgo.MessageReaction) {
	log.Printf("discord: handling reaction %s %s on message %q from user %s", kind, r.Emoji.Name, r.MessageID, r.UserID)

	msg, err := s.ChannelMessage(r.ChannelID, r.MessageID)
	if err != nil {
		log.Printf("discord: error fetching message %q", r.MessageID)
		return
	}

	for _, handler := range c.reactionHandlers {
		err := (*handler)(s, r, msg)
		if err != nil {
			log.Printf("discord: error handling reaction %s %s on message %q: %s", kind, r.Emoji.Name, r.MessageID, spew.Sdump(err))
		}
	}
}
