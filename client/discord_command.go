package client

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
)

type DiscordCommand struct {
	Handler            DiscordCommandHandler
	ApplicationCommand *discordgo.ApplicationCommand
	Async              bool
}

type DiscordCommandInput struct {
	IC         *discordgo.InteractionCreate
	User       *discordgo.User
	Slug       string // command and subcommand, for logging
	Command    string
	Subcommand *discordgo.ApplicationCommandInteractionDataOption
}

type DiscordCommandHandler func(*discordgo.Session, *DiscordCommandInput) (string, error)

func (c *Discord) AddCommand(command *DiscordCommand) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.commandsRegistered {
		panic("can't call AddCommand() after RegisterCommands()")
	}
	c.appCommandHandlers[command.ApplicationCommand.Name] = command
}

func (c *Discord) RegisterCommands() error {
	var appCommands []*discordgo.ApplicationCommand
	for _, command := range c.appCommandHandlers {
		appCommands = append(appCommands, command.ApplicationCommand)
	}
	_, err := c.s.ApplicationCommandBulkOverwrite(c.s.State.User.ID, c.Guild.ID, appCommands)
	if err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.commandsRegistered = true

	return nil
}

func (c *Discord) commandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		log.Printf("discord: ignoring interaction of unknown type: %v", i.Type)
		return
	}

	input := &DiscordCommandInput{
		IC:      i,
		User:    i.User,
		Command: i.ApplicationCommandData().Name,
	}
	if input.User == nil {
		input.User = i.Member.User
	}

	for _, opt := range i.ApplicationCommandData().Options {
		if opt.Type == discordgo.ApplicationCommandOptionSubCommand {
			input.Subcommand = opt
		}
	}
	command, ok := c.appCommandHandlers[input.Command]
	if !ok {
		log.Printf("discord: received unknown command: %#v %#v", i, i.ApplicationCommandData())
		return
	}

	input.Slug = input.Command
	if input.Subcommand != nil {
		input.Slug += " " + input.Subcommand.Name
	}
	log.Printf("discord: handling command %q from @%s", input.Slug, input.User.Username)

	// For async handlers, acknowledge the interaction immediately. This means
	// we can take more than 3 seconds in Handler(). (If we don't do this,
	// Discord will report an error to the user.)
	if command.Async {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			log.Printf("discord: error acknowledging command %q: %s", input.Slug, spew.Sdump(err))
			return
		}
	}

	// Call the handler!
	reply, err := command.Handler(s, input)
	if err != nil {
		log.Printf("discord: error handling command %q: %s", input.Slug, spew.Sdump(err))
		reply = fmt.Sprintf("🚨 Bot Error! Please ping in %s for help.\n```\n%s\n```", c.TechChannel.Mention(), spew.Sdump(err))
	}

	if command.Async {
		_, err = s.InteractionResponseEdit(
			input.IC.Interaction, &discordgo.WebhookEdit{
				Content: &reply,
			},
		)
	} else {
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: reply},
		})
	}
	if err != nil {
		log.Printf("discord: error responding to command %q: %s", input.Slug, spew.Sdump(err))
	}
}

func (c *Discord) OptionByName(options []*discordgo.ApplicationCommandInteractionDataOption, name string) (*discordgo.ApplicationCommandInteractionDataOption, error) {
	var result *discordgo.ApplicationCommandInteractionDataOption
	for _, opt := range options {
		if opt.Name == name {
			result = opt
		}
	}
	if result == nil {
		return nil, fmt.Errorf("could not find option %q in options list", name)
	}
	return result, nil
}
