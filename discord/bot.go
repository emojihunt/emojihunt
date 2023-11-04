package discord

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
)

const DefaultHandlerTimeout = 120 * time.Second

type Bot interface {
	Register() (ac *discordgo.ApplicationCommand, async bool)
	Handle(*discordgo.Session, *CommandInput) (string, error)
}

type CommandInput struct {
	IC         *discordgo.InteractionCreate
	User       *discordgo.User
	Slug       string // command and subcommand, for logging
	Command    string
	Subcommand *discordgo.ApplicationCommandInteractionDataOption
}

type commandHandler struct {
	ApplicationCommand *discordgo.ApplicationCommand
	Async              bool
	Handler            func(*discordgo.Session, *CommandInput) (string, error)
}

func (c *Client) RegisterBots(bots ...Bot) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.commandsRegistered {
		panic("RegisterBots() was called twice")
	}
	c.commandsRegistered = true

	// Call each bot's Register() method
	var appCommands []*discordgo.ApplicationCommand
	for _, bot := range bots {
		ac, async := bot.Register()
		if _, ok := c.appCommandHandlers[ac.Name]; ok {
			panic("duplicate app command: " + ac.Name)
		}
		appCommands = append(appCommands, ac)
		c.appCommandHandlers[ac.Name] = &commandHandler{
			Handler:            bot.Handle,
			ApplicationCommand: ac,
			Async:              async,
		}
	}

	// Send list of registrations to Discord
	_, err := c.s.ApplicationCommandBulkOverwrite(c.s.State.User.ID, c.Guild.ID, appCommands)
	if err != nil {
		panic(err)
	}
}

func (c *Client) OptionByName(options []*discordgo.ApplicationCommandInteractionDataOption, name string) (*discordgo.ApplicationCommandInteractionDataOption, error) {
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

func (c *Client) commandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionApplicationCommand {
		log.Printf("discord: ignoring interaction of unknown type: %v", i.Type)
		return
	}

	input := &CommandInput{
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
		reply = fmt.Sprintf("```*** 🚨 BOT ERROR ***\n\n%s\nPlease ping in #%s for help.\n```", spew.Sdump(err), c.TechChannel.Name)
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
