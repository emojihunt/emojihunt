package client

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
)

type DiscordCommand struct {
	InteractionType discordgo.InteractionType
	Handler         DiscordCommandHandler

	ApplicationCommand *discordgo.ApplicationCommand // for InteractionApplicationCommand
	CustomID           string                        // for InteractionMessageComponent
}

type DiscordCommandInput struct {
	IC         *discordgo.InteractionCreate
	User       *discordgo.User
	Command    string
	Subcommand *discordgo.ApplicationCommandInteractionDataOption
}

type DiscordCommandHandler func(*discordgo.Session, *DiscordCommandInput) (string, error)

const DiscordMagicReplyDefer = "$DEFER$"

func (c *Discord) RegisterCommands(commands []*DiscordCommand) error {
	var appCommands []*discordgo.ApplicationCommand
	for _, command := range commands {
		switch command.InteractionType {
		case discordgo.InteractionApplicationCommand:
			appCommands = append(appCommands, command.ApplicationCommand)
			c.appCommandHandlers[command.ApplicationCommand.Name] = command.Handler
		case discordgo.InteractionMessageComponent:
			c.componentHandlers[command.CustomID] = command.Handler
		default:
			return fmt.Errorf("unknown interaction type: %v", command.InteractionType)
		}
	}
	_, err := c.s.ApplicationCommandBulkOverwrite(c.s.State.User.ID, c.GuildID, appCommands)
	return err
}

func (c *Discord) commandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var idesc string
	var handler DiscordCommandHandler
	var ok bool
	input := &DiscordCommandInput{
		IC:   i,
		User: i.User,
	}
	if input.User == nil {
		input.User = i.Member.User
	}

	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		idesc = "application command"
		input.Command = i.ApplicationCommandData().Name
		for _, opt := range i.ApplicationCommandData().Options {
			if opt.Type == discordgo.ApplicationCommandOptionSubCommand {
				input.Subcommand = opt
			}
		}
		if handler, ok = c.appCommandHandlers[input.Command]; !ok {
			log.Printf("discord: received unknown %s: %#v %#v", idesc, i, i.ApplicationCommandData())
			return
		}
	case discordgo.InteractionMessageComponent:
		idesc = "component interaction"
		input.Command = i.MessageComponentData().CustomID
		parts := strings.Split(input.Command, "/")
		if handler, ok = c.componentHandlers[parts[0]]; !ok {
			log.Printf("discord: received unknown %s: %#v %#v", idesc, i, i.MessageComponentData())
			return
		}
	default:
		log.Printf("discord: ignoring interaction of unknown type: %v", i.Type)
		return
	}

	var cmdName = input.Command
	if input.Subcommand != nil {
		cmdName += " " + input.Subcommand.Name
	}
	log.Printf("discord: handling %s %q from @%s", idesc, cmdName, input.User.Username)

	// Call the handler! We need to run our logic and call
	// InteractionRespond within 3 seconds, otherwise Discord will report an
	// error to the user.
	reply, err := handler(s, input)
	if err != nil {
		log.Printf("discord: error handling %s %q: %s", idesc, input.Command, spew.Sdump(err))
		reply = fmt.Sprintf("🚨 Bot Error! Please ping in %s for help.\n```\n%s\n```", c.TechChannel.Mention(), spew.Sdump(err))
	}

	if reply == DiscordMagicReplyDefer {
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
	} else {
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: reply},
		})
	}
	if err != nil {
		log.Printf("discord: error responding to %s %q: %s", idesc, input.Command, spew.Sdump(err))
	}
}
