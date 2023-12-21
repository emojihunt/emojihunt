package discord

import (
	"context"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/xerrors"
)

type Bot interface {
	Register() (cmd *discordgo.ApplicationCommand, async bool)
	Handle(context.Context, *CommandInput) (string, error)
	HandleScheduledEvent(context.Context, *discordgo.GuildScheduledEventUpdate) error
}

const DefaultHandlerTimeout = 120 * time.Second

type CommandInput struct {
	client *Client

	IC         *discordgo.InteractionCreate
	User       *discordgo.User
	Command    string
	Subcommand *discordgo.ApplicationCommandInteractionDataOption
}

func (i CommandInput) EditMessage(msg string) error {
	_, err := i.client.s.InteractionResponseEdit(
		i.IC.Interaction,
		&discordgo.WebhookEdit{Content: &msg},
	)
	if err != nil {
		return xerrors.Errorf("InteractionResponseEdit: %w", err)
	}
	return nil
}

type botRegistration struct {
	Name               string
	ApplicationCommand *discordgo.ApplicationCommand
	Async              bool
	Bot
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
		if _, ok := c.botsByCommand[ac.Name]; ok {
			panic("duplicate app command: " + ac.Name)
		}
		appCommands = append(appCommands, ac)
		c.botsByCommand[ac.Name] = &botRegistration{
			Name:               ac.Name,
			ApplicationCommand: ac,
			Async:              async,
			Bot:                bot,
		}
	}

	// Send list of registrations to Discord
	_, err := c.s.ApplicationCommandBulkOverwrite(c.s.State.User.ID, c.Guild.ID, appCommands)
	if err != nil {
		panic(err)
	}
}

func (c *Client) OptionByName(
	options []*discordgo.ApplicationCommandInteractionDataOption, name string,
) (*discordgo.ApplicationCommandInteractionDataOption, bool) {
	var result *discordgo.ApplicationCommandInteractionDataOption
	for _, opt := range options {
		if opt.Name == name {
			result = opt
		}
	}
	if result == nil {
		return nil, false
	}
	return result, true
}
