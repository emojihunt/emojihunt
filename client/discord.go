package client

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
)

type DiscordConfig struct {
	GuildID                                                                     string
	QMChannelName, GeneralChannelName, StatusUpdateChannelName, TechChannelName string
	PuzzleCategoryName, SolvedCategoryName                                      string
	QMRoleName                                                                  string
}

type Discord struct {
	s       *discordgo.Session
	GuildID string
	// The QM channel contains a central log of interesting bot actions, as well as the only place for
	// advanced bot usage, such as puzzle or round creation.
	QMChannel *discordgo.Channel
	// The general channel has all users, and has announcements from the bot.
	GeneralChannel *discordgo.Channel
	// The channel in which to post status updates.
	StatusUpdateChannel *discordgo.Channel
	// The tech channel has error messages.
	TechChannel *discordgo.Channel
	// The puzzle channel category.
	PuzzleCategory *discordgo.Channel
	// The category for solved puzzles.
	SolvedCategory *discordgo.Channel
	// The Role ID for the QM role.
	QMRoleID string

	handlers map[string]DiscordCommandHandler
}

func NewDiscord(token string, c DiscordConfig) (*Discord, error) {
	// Initialize discordgo client
	s, err := discordgo.New(token)
	if err != nil {
		return nil, err
	}
	s.Identify.Intents = discordgo.IntentsGuildMessages
	if err := s.Open(); err != nil {
		return nil, err
	}

	// Map channel/etc. names in config to actual Discord API IDs
	chs, err := s.GuildChannels(c.GuildID)
	if err != nil {
		return nil, fmt.Errorf("error creating channel ID cache: %v", err)
	}
	var qm, puz, ar, gen, st, tech *discordgo.Channel
	for _, ch := range chs {
		switch ch.Name {
		case c.QMChannelName:
			qm = ch
		case c.PuzzleCategoryName:
			puz = ch
		case c.SolvedCategoryName:
			ar = ch
		case c.GeneralChannelName:
			gen = ch
		case c.StatusUpdateChannelName:
			st = ch
		case c.TechChannelName:
			tech = ch
		}
	}
	roles, err := s.GuildRoles(c.GuildID)
	if err != nil {
		return nil, fmt.Errorf("error fetching roles: %v", err)
	}
	var qmRoleID string
	for _, r := range roles {
		if r.Name == c.QMRoleName {
			qmRoleID = r.ID
			break
		}
	}
	if qmRoleID == "" {
		return nil, fmt.Errorf("QM role %q not found in roles: %v", c.QMRoleName, roles)
	}

	// Set up slash commands; return
	commandHandlers := make(map[string]DiscordCommandHandler)
	discord := &Discord{
		s:                   s,
		GuildID:             c.GuildID,
		QMChannel:           qm,
		GeneralChannel:      gen,
		StatusUpdateChannel: st,
		TechChannel:         tech,
		PuzzleCategory:      puz,
		SolvedCategory:      ar,
		QMRoleID:            qmRoleID,
		handlers:            commandHandlers,
	}
	s.AddHandler(discord.commandHandler)

	return discord, nil
}

func (c *Discord) Close() error {
	return c.s.Close()
}

func (c *Discord) ChannelSend(ch *discordgo.Channel, msg string) error {
	_, err := c.s.ChannelMessageSend(ch.ID, msg)
	return err
}

func (c *Discord) ChannelSendEmbed(ch *discordgo.Channel, embed *discordgo.MessageEmbed) error {
	_, err := c.s.ChannelMessageSendEmbed(ch.ID, embed)
	return err
}

func (c *Discord) ChannelSendRawID(chID, msg string) error {
	_, err := c.s.ChannelMessageSend(chID, msg)
	return err
}

func (c *Discord) CreateChannel(name string) (*discordgo.Channel, error) {
	return c.s.GuildChannelCreateComplex(c.GuildID, discordgo.GuildChannelCreateData{
		Name:     name,
		Type:     discordgo.ChannelTypeGuildText,
		ParentID: c.PuzzleCategory.ID,
	})
}

func (c *Discord) SetChannelName(chID, name string) error {
	_, err := c.s.ChannelEdit(chID, name)
	return err
}

func (c *Discord) SetChannelCategory(chID string, category *discordgo.Channel) error {
	ch, err := c.s.Channel(chID)
	if err != nil {
		return fmt.Errorf("channel id %s not found", chID)
	}

	if ch.ParentID == category.ID {
		return nil // no-op
	}

	_, err = c.s.ChannelEditComplex(chID, &discordgo.ChannelEdit{
		ParentID:             category.ID,
		PermissionOverwrites: category.PermissionOverwrites,
	})
	if err != nil {
		return fmt.Errorf("error moving channel to category %q: %v", category.Name, err)
	}
	return nil
}

// Set the pinned status message, by posting one or editing the existing one.
// No-op if the status was already set.
func (c *Discord) CreateUpdatePin(chanID, header string, embed *discordgo.MessageEmbed) error {
	existing, err := c.s.ChannelMessagesPinned(chanID)
	if err != nil {
		return err
	}
	var statusMessage *discordgo.Message
	for _, msg := range existing {
		if len(msg.Embeds) > 0 && msg.Embeds[0].Author.Name == header {
			if statusMessage != nil {
				log.Printf("discord: multiple status messages in %v, editing last one", chanID)
			}
			statusMessage = msg
		}
	}

	if statusMessage == nil {
		// create a pinned message
		m, err := c.s.ChannelMessageSendEmbed(chanID, embed)
		if err != nil {
			return err
		}
		return c.s.ChannelMessagePin(chanID, m.ID)
	} else {
		// update existing pinned message
		if statusMessage.Embeds[0] == embed {
			return nil // no-op
		}
		_, err = c.s.ChannelMessageEditEmbed(chanID, statusMessage.ID, embed)
		return err
	}
}
