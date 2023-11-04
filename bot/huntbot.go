package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/discovery"
	"github.com/emojihunt/emojihunt/schema"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
)

type HuntBot struct {
	main      context.Context
	db        *db.Client
	discord   *discord.Client
	discovery *discovery.Poller
	syncer    *syncer.Syncer
	state     *state.State
}

const fullResyncTimeout = 60 * time.Minute

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
			{
				Name:        "yikes",
				Description: "Force re-sync all puzzles 🔨",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "confirm",
						Description: "Please enter ⚠️ to confirm this operation",
						Type:        discordgo.ApplicationCommandOptionString,
						Required:    true,
					},
				},
			},
		},
	}, false
}

func (b *HuntBot) Handle(ctx context.Context, s *discordgo.Session,
	i *discord.CommandInput) (string, error) {

	if i.IC.ChannelID != b.discord.QMChannel.ID &&
		i.IC.ChannelID != b.discord.TechChannel.ID {
		return fmt.Sprintf(
			":tv: Please use `/huntbot` commands in the %s or %s channel.",
			b.discord.QMChannel.Mention(),
			b.discord.TechChannel.Mention(),
		), nil
	}

	b.state.Lock()
	defer b.state.CommitAndUnlock()

	var reply string
	switch i.Subcommand.Name {
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
	case "yikes":
		if confirmOpt, err := b.discord.OptionByName(i.Subcommand.Options, "confirm"); err != nil {
			return "", err
		} else if value := confirmOpt.StringValue(); value != "⚠️" {
			return fmt.Sprintf(":no_smoking: Incorrect confirmation value %q, operation aborted.",
				value), nil
		}
		go b.fullResync(s, i)
		return ":warning: Initiated full re-sync!", nil
	default:
		return "", fmt.Errorf("unexpected /huntbot subcommand: %q", i.Subcommand.Name)
	}
}

func (b *HuntBot) fullResync(s *discordgo.Session, i *discord.CommandInput) {
	ctx, cancel := context.WithTimeout(b.main, fullResyncTimeout)
	defer func() {
		if err := recover(); err != nil {
			log.Printf("RestorePlaceholderEvent: %v", err)
		}
		cancel()
	}()

	var errs = make(map[string]error)

	puzzles, err := b.db.ListPuzzles(ctx)
	if err == nil {
		for j, id := range puzzles {
			if b.state.IsKilled() {
				err = fmt.Errorf("huntbot is disabled")
				break
			}

			var puzzle *schema.Puzzle
			puzzle, err = b.db.LockByID(ctx, id)
			if err != nil {
				err = fmt.Errorf("failed to load %q: %s", id, spew.Sdump(err))
				break
			}

			_, err = b.syncer.ForceUpdate(ctx, puzzle)
			if err != nil {
				log.Printf("huntbot yikes: re-sync err in %q: %v", puzzle.Name, spew.Sdump(err))
				errs[puzzle.Name] = err
			}
			puzzle.Unlock()

			if j%10 == 0 {
				msg := fmt.Sprintf(
					":warning: Initiated full re-sync! (%d / %d)", j, len(puzzles),
				)
				_, err = s.InteractionResponseEdit(
					i.IC.Interaction, &discordgo.WebhookEdit{
						Content: &msg,
					},
				)
				if err != nil {
					err = fmt.Errorf("failed to update with progress: %v", err)
					break
				}
			}
		}
	}

	var msg string
	if err != nil {
		log.Printf("huntbot yikes: failed to re-sync: %v", err)
		msg = fmt.Sprintf("```*** ⚠️ FULL RE-SYNC FAILED ***\n\n%s```", spew.Sdump(err))
	} else if len(errs) > 0 {
		msg = "```*** FULL RE-SYNC COMPLETED WITH ERRORS ***\n\n"
		for name, err := range errs {
			msg += fmt.Sprintf("%s: %s\n", strings.ToUpper(name), spew.Sdump(err))
		}
	} else {
		log.Printf("huntbot yikes: completed successfully")
		msg = ":recycle: Full re-sync completed successfully!"
	}
	_, err = s.InteractionResponseEdit(
		i.IC.Interaction, &discordgo.WebhookEdit{
			Content: &msg,
		},
	)
	if err != nil {
		log.Printf("huntbot yikes: failed to update with status %q: %v", msg, err)
	}
}
