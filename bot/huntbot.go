package bot

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/discovery"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
	"golang.org/x/xerrors"
)

type HuntBot struct {
	main      context.Context
	discord   *discord.Client
	discovery *discovery.Poller
	state     *state.Client
	syncer    *syncer.Syncer
}

func NewHuntBot(main context.Context, discord *discord.Client,
	discovery *discovery.Poller, state *state.Client, syncer *syncer.Syncer) discord.Bot {
	return &HuntBot{main, discord, discovery, state, syncer}
}

func (b *HuntBot) Register() (*discordgo.ApplicationCommand, bool) {
	return &discordgo.ApplicationCommand{
		Name:        "huntbot",
		Description: "Robot control panel 🤖",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "kill",
				Description: "Temporarily disable Huntbot ✋",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "enable",
				Description: "Re-enable Huntbot 📡",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}, false
}

func (b *HuntBot) Handle(ctx context.Context, input *discord.CommandInput) (string, error) {
	if input.IC.ChannelID != b.discord.QMChannel.ID {
		return fmt.Sprintf(
			":tv: Please use `/huntbot` commands in the %s channel.",
			b.discord.QMChannel.Mention(),
		), nil
	}

	var reply string
	// TODO: there's a race condition here:
	var wasKilled = b.state.IsDisabled(ctx)
	switch input.Subcommand.Name {
	case "kill":
		if !wasKilled {
			b.state.DisableHuntbot(ctx, true)
			reply = "Ok, I've disabled the bot for now.  Enable it with `/huntbot enable`."
		} else {
			reply = "The bot was already disabled. Enable it with `/huntbot enable`."
		}
		b.discord.UpdateStatus(ctx) // best-effort, ignore errors
		return reply, nil
	case "enable":
		if !wasKilled {
			b.state.DisableHuntbot(ctx, false)
			reply = "Ok, I've enabled the bot for now. Disable it with `/huntbot kill`."
		} else {
			reply = "The bot was already enabled. Disable it with `/huntbot kill`."
		}
		b.discord.UpdateStatus(ctx) // best-effort, ignore errors
		return reply, nil
	default:
		return "", xerrors.Errorf("unexpected /huntbot subcommand: %q", input.Subcommand.Name)
	}
}

func (b *HuntBot) HandleScheduledEvent(context.Context,
	*discordgo.GuildScheduledEventUpdate) error {
	return nil
}
