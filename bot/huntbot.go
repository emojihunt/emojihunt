package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/discovery"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
	"golang.org/x/xerrors"
)

type HuntBot struct {
	main      context.Context
	db        *db.Client
	discord   *discord.Client
	discovery *discovery.Poller
	syncer    *syncer.Syncer
	state     *state.State
}

func NewHuntBot(main context.Context, db *db.Client, discord *discord.Client,
	discovery *discovery.Poller, syncer *syncer.Syncer, state *state.State) discord.Bot {
	return &HuntBot{main, db, discord, discovery, syncer, state}
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
	if input.IC.ChannelID != b.discord.QMChannel.ID &&
		input.IC.ChannelID != b.discord.TechChannel.ID {
		return fmt.Sprintf(
			":tv: Please use `/huntbot` commands in the %s or %s channel.",
			b.discord.QMChannel.Mention(),
			b.discord.TechChannel.Mention(),
		), nil
	}

	b.state.Lock()
	defer b.state.CommitAndUnlock()

	var reply string
	switch input.Subcommand.Name {
	case "kill":
		if !b.state.HuntbotDisabled {
			b.state.HuntbotDisabled = true
			reply = "Ok, I've disabled the bot for now.  Enable it with `/huntbot enable`."
		} else {
			reply = "The bot was already disabled. Enable it with `/huntbot enable`."
		}
		b.discovery.CancelAllRoundCreation()
		b.discord.UpdateStatus(b.state) // best-effort, ignore errors
		return reply, nil
	case "enable":
		if b.state.HuntbotDisabled {
			b.state.HuntbotDisabled = false
			reply = "Ok, I've enabled the bot for now. Disable it with `/huntbot kill`."
		} else {
			reply = "The bot was already enabled. Disable it with `/huntbot kill`."
		}
		go func() {
			defer func() {
				if err := recover(); err != nil {
					log.Printf("InitializeRoundCreation: %v", err)
				}
			}()

			// Will block until we release the state lock
			b.discovery.InitializeRoundCreation() // TODO: factor out
		}()
		b.discord.UpdateStatus(b.state) // best-effort, ignore errors
		return reply, nil
	default:
		return "", xerrors.Errorf("unexpected /huntbot subcommand: %q", input.Subcommand.Name)
	}
}

func (b *HuntBot) HandleScheduledEvent(context.Context,
	*discordgo.GuildScheduledEventUpdate) error {
	return nil
}
