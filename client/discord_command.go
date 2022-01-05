package client

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
)

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
