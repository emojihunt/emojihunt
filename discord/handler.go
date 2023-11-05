package discord

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/getsentry/sentry-go"
)

const HandlerTimeout = 120 * time.Second

func WrapHandler[Action any](
	main context.Context,
	name string,
	handle func(context.Context, Action) error,
) func(*discordgo.Session, Action) {
	return func(s *discordgo.Session, a Action) {
		ctx, cancel := context.WithTimeout(main, HandlerTimeout)
		defer cancel()

		hub := sentry.CurrentHub().Clone()
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("task", name)
		})
		ctx = sentry.SetHubOnContext(ctx, hub)
		defer sentry.RecoverWithContext(ctx)

		if err := handle(ctx, a); err != nil {
			sentry.GetHubFromContext(ctx).CaptureException(err)
		}
	}
}

func (c *Client) HandleApplicationCommand(ctx context.Context,
	i *discordgo.InteractionCreate) error {

	if i.Type != discordgo.InteractionApplicationCommand {
		return nil
	}

	input := &CommandInput{
		client:  c,
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
		return fmt.Errorf("unknown command %q", input.Command)
	}

	var task = fmt.Sprintf("bot.%s", input.Command)
	if input.Subcommand != nil {
		task = fmt.Sprintf("%s.%s", task, input.Subcommand.Name)
	}
	log.Printf("handling command %s from @%s", task, input.User.Username)
	sentry.GetHubFromContext(ctx).ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", task)
	})

	// For async handlers, acknowledge the interaction immediately. This means
	// we can take more than 3 seconds in Handler(). (If we don't do this,
	// Discord will report an error to the user.)
	if command.Async {
		err := c.s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			return err
		}
	}

	// Call the handler!
	reply, err := command.Handler(ctx, input)
	if err != nil {
		var url string
		event := sentry.GetHubFromContext(ctx).CaptureException(err)
		if event != nil {
			url = fmt.Sprintf(c.issueURL, *event)
		}
		reply = fmt.Sprintf("🚨 Error! Please ping in %s for help. %s",
			c.TechChannel.Mention(), url)
	}

	if command.Async {
		err := input.EditMessage(reply)
		return err
	} else {
		err := c.s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: reply},
		})
		return err
	}
}
