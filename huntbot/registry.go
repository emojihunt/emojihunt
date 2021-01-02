package huntbot

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/discord"
)

var discordHandlers = []discord.NewMessageHandler{
	testHandler,
}

func testHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore self created messages.
	if m.Author.ID == s.State.User.ID {
		return
	}
	log.Printf("processing message: %v", m.Content)
	s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("echo: %v", m.Content))
}
