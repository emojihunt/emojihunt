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

	// For Message Components: disable non-link buttons after the first click
	OnlyOnce bool
}

type DiscordCommandInput struct {
	IC         *discordgo.InteractionCreate
	User       *discordgo.User
	Slug       string // command and subcommand, for logging
	Command    string
	Subcommand *discordgo.ApplicationCommandInteractionDataOption
}

type DiscordCommandHandler func(*discordgo.Session, *DiscordCommandInput) (string, error)

const discordMagicReplyDefer = "$DEFER$"

func (c *Discord) RegisterCommands(commands []*DiscordCommand) error {
	var appCommands []*discordgo.ApplicationCommand
	for _, command := range commands {
		switch command.InteractionType {
		case discordgo.InteractionApplicationCommand:
			appCommands = append(appCommands, command.ApplicationCommand)
			c.appCommandHandlers[command.ApplicationCommand.Name] = command
		case discordgo.InteractionMessageComponent:
			c.componentHandlers[command.CustomID] = command
		default:
			return fmt.Errorf("unknown interaction type: %v", command.InteractionType)
		}
	}
	_, err := c.s.ApplicationCommandBulkOverwrite(c.s.State.User.ID, c.Guild.ID, appCommands)
	return err
}

func (c *Discord) commandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var idesc string
	var command *DiscordCommand
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
		if command, ok = c.appCommandHandlers[input.Command]; !ok {
			log.Printf("discord: received unknown %s: %#v %#v", idesc, i, i.ApplicationCommandData())
			return
		}
	case discordgo.InteractionMessageComponent:
		idesc = "component interaction"
		input.Command = i.MessageComponentData().CustomID
		parts := strings.Split(input.Command, "/")
		if command, ok = c.componentHandlers[parts[0]]; !ok {
			log.Printf("discord: received unknown %s: %#v %#v", idesc, i, i.MessageComponentData())
			return
		}
	default:
		log.Printf("discord: ignoring interaction of unknown type: %v", i.Type)
		return
	}

	input.Slug = input.Command
	if input.Subcommand != nil {
		input.Slug += " " + input.Subcommand.Name
	}
	log.Printf("discord: handling %s %q from @%s", idesc, input.Slug, input.User.Username)
	if command.OnlyOnce {
		if err := c.enableMessageComponents(s, input.IC.Message, false); err != nil {
			log.Printf("discord: error disabling message components for %q: %v", input.Slug, err)
			return // time out; user will see "interaction failed" message
		}
	}

	// Call the handler! We need to run our logic and call
	// InteractionRespond within 3 seconds, otherwise Discord will report an
	// error to the user.
	reply, err := command.Handler(s, input)
	if err != nil {
		if command.OnlyOnce {
			if err := c.enableMessageComponents(s, input.IC.Message, true); err != nil {
				log.Printf("discord: error reenabling message components for %q: %v", input.Slug, err)
			}
		}
		log.Printf("discord: error handling %s %q: %s", idesc, input.Slug, spew.Sdump(err))
		reply = fmt.Sprintf("🚨 Bot Error! Please ping in %s for help.\n```\n%s\n```", c.TechChannel.Mention(), spew.Sdump(err))
	}

	if reply == discordMagicReplyDefer {
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
		log.Printf("discord: error responding to %s %q: %s", idesc, input.Slug, spew.Sdump(err))
	}
}

// When performing a long-running action in response to a Discord interaction
// (slash command, message button), we have to send our initial response within
// 3 seconds and defer the rest of the work to complete asynchronously. To do
// this:
//
//   return ReplyAsync(s, i, func() (string, error) { ... })
//
// Discord will display "huntbot is thinking..." until the async function
// completes.
//
func (d *Discord) ReplyAsync(s *discordgo.Session, i *DiscordCommandInput, fn func() (string, error)) (string, error) {
	go func() {
		reply, err := fn()
		if err != nil {
			if err := d.enableMessageComponents(s, i.IC.Message, true); err != nil {
				log.Printf("discord: error reenabling message components for %q: %v", i.Slug, err)
			}
			log.Printf("discord: error handling interaction %q: %s", i.Slug, spew.Sdump(err))
			reply = fmt.Sprintf("🚨 Bot Error! Please ping in %s for help.\n```\n%s\n```", d.TechChannel.Mention(), spew.Sdump(err))
		}
		_, err = s.InteractionResponseEdit(
			s.State.User.ID, i.IC.Interaction, &discordgo.WebhookEdit{
				Content: reply,
			},
		)
		if err != nil {
			log.Printf("discord: error responding to interaction %q: %s", i.Slug, spew.Sdump(err))
		} else {
			log.Printf("discord: finished async processing for interaction %q", i.Slug)
		}
	}()
	return discordMagicReplyDefer, nil
}

func (d *Discord) OptionByName(options []*discordgo.ApplicationCommandInteractionDataOption, name string) (*discordgo.ApplicationCommandInteractionDataOption, error) {
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

func (d *Discord) enableMessageComponents(s *discordgo.Session, message *discordgo.Message, enabled bool) error {
	if len(message.Components) < 1 {
		return nil
	}

	var result []discordgo.MessageComponent
	for _, component := range message.Components {
		if component.Type() != discordgo.ActionsRowComponent {
			return fmt.Errorf("expected only actions rows at top level in %#v", message.Components)
		}
		rewritten := *(component.(*discordgo.ActionsRow))
		rewritten.Components = make([]discordgo.MessageComponent, 0)
		for _, item := range component.(*discordgo.ActionsRow).Components {
			switch item.Type() {
			case discordgo.ActionsRowComponent:
				return fmt.Errorf("unexpected nested actions row in %#v", message.Components)
			case discordgo.ButtonComponent:
				moditem := *(item.(*discordgo.Button))
				if moditem.Style != discordgo.LinkButton {
					moditem.Disabled = !enabled
				}
				rewritten.Components = append(rewritten.Components, moditem)
			case discordgo.SelectMenuComponent:
				moditem := *(item.(*discordgo.SelectMenu))
				moditem.Options = make([]discordgo.SelectMenuOption, 0)
				rewritten.Components = append(rewritten.Components, moditem)
			default:
				return fmt.Errorf("unexpected message component type %v", item.Type())
			}
		}
		result = append(result, rewritten)
	}

	edit := discordgo.NewMessageEdit(message.ChannelID, message.ID)
	edit.Components = result
	_, err := s.ChannelMessageEditComplex(edit)
	return err
}
