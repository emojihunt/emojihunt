package client

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
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

func NewDiscord(s *discordgo.Session, c DiscordConfig) (*Discord, error) {
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

//
// Helpers for Slash Commands (bots)
//

type DiscordCommand struct {
	ApplicationCommand *discordgo.ApplicationCommand
	Handler            DiscordCommandHandler
}

type DiscordCommandInput struct {
	IC         *discordgo.InteractionCreate
	User       *discordgo.User
	Command    string
	Subcommand *discordgo.ApplicationCommandInteractionDataOption
}

type DiscordCommandHandler func(*discordgo.Session, *DiscordCommandInput) (string, error)

func (c *Discord) RegisterCommands(commands []*DiscordCommand) error {
	var appCommands []*discordgo.ApplicationCommand
	for _, command := range commands {
		appCommands = append(appCommands, command.ApplicationCommand)
		c.handlers[command.ApplicationCommand.Name] = command.Handler
	}
	_, err := c.s.ApplicationCommandBulkOverwrite(c.s.State.User.ID, c.GuildID, appCommands)
	return err
}

func (c *Discord) commandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	input := &DiscordCommandInput{
		IC:         i,
		User:       i.User,
		Command:    i.ApplicationCommandData().Name,
		Subcommand: nil,
	}
	if input.User == nil {
		input.User = i.Member.User
	}
	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Type == discordgo.ApplicationCommandOptionSubCommand {
			input.Subcommand = opt
		}
	}

	if handler, ok := c.handlers[input.Command]; ok {
		var cmdName = input.Command
		if input.Subcommand != nil {
			cmdName += " " + input.Subcommand.Name
		}
		log.Printf("discord: handling command %q from @%s", cmdName, input.User.Username)

		// Call the handler! We need to run our logic and call
		// InteractionRespond within 3 seconds, otherwise Discord will report an
		// error to the user.
		reply, err := handler(s, input)
		if err != nil {
			log.Printf("discord: error handling interaction %q: %s", input.Command, spew.Sdump(err))
			reply = fmt.Sprintf("```\n🚨 Bot Error\n%s\n```", spew.Sdump(err))
		}

		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: reply},
		})
		if err != nil {
			log.Printf("discord: error responding to interaction %q: %s", input.Command, spew.Sdump(err))
		}
	} else {
		log.Printf("discord: received unknown interaction: %#v %#v", i, i.ApplicationCommandData())
	}
}
