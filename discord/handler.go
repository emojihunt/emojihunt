package discord

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/getsentry/sentry-go"
	"golang.org/x/xerrors"
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

// Application Command Handling

func (c *Client) handleCommand(
	ctx context.Context, i *discordgo.InteractionCreate,
) error {
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
	command, ok := c.botsByCommand[input.Command]
	if !ok {
		return xerrors.Errorf("unknown command %q", input.Command)
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
			return xerrors.Errorf("InteractionRespond: %w", err)
		}
	}

	// Call the handler!
	reply, err := command.Handle(ctx, input)
	if err != nil {
		var url string
		event := sentry.GetHubFromContext(ctx).CaptureException(
			xerrors.Errorf("%s.Handle: %w", command.Name, err),
		)
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
		if err != nil {
			return xerrors.Errorf("InteractionRespond: %w", err)
		}
		return nil
	}
}

// Scheduled Event Handling

func (c *Client) handleScheduledEvent(
	ctx context.Context, e *discordgo.GuildScheduledEventUpdate,
) error {
	hub := sentry.GetHubFromContext(ctx)
	for _, bot := range c.botsByCommand {
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetTag("task", fmt.Sprintf("bot.%s.event", bot.Name))
		})
		err := bot.HandleScheduledEvent(ctx, e)
		if err != nil {
			return xerrors.Errorf("%s.HandleScheduledEvent: %w", bot.Name, err)
		}
	}
	return nil
}

// Reaction Handling

func (c *Client) RegisterReactionHandler(
	handler func(context.Context, *discordgo.MessageReaction) error,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.reactionHandlers = append(c.reactionHandlers, &handler)
}

func (c *Client) handleReaction(
	ctx context.Context, r *discordgo.MessageReaction,
) error {
	for _, handler := range c.reactionHandlers {
		err := (*handler)(ctx, r)
		if err != nil {
			return xerrors.Errorf("reaction handler: %w", err)
		}
	}
	return nil
}

// Rate Limit Tracking

func (c *Client) CheckRateLimit(url string) *time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	limit := c.rateLimits[url]
	if limit == nil || time.Now().After(*limit) {
		return nil
	}
	return limit
}

func (c *Client) handleRateLimit(
	ctx context.Context, r *discordgo.RateLimit,
) error {
	if strings.HasSuffix(r.URL, "/commands") {
		// If we restart the bot too many times in a row, we'll get
		// rate-limited on the Register Application Commands endpoint. Just
		// ignore.
		log.Printf("rate-limited when re-registering application commands")
		return nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	expiry := time.Now().Add(r.TooManyRequests.RetryAfter)
	c.rateLimits[r.URL] = &expiry

	wait := time.Until(expiry).Round(time.Second)
	return xerrors.Errorf("discord rate limit: %s (wait %s)", r.URL, wait)
}
