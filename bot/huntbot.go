package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/client"
	"github.com/emojihunt/emojihunt/database"
	"github.com/emojihunt/emojihunt/discovery"
	"github.com/emojihunt/emojihunt/schema"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
)

func RegisterHuntbotCommand(ctx context.Context, airtable *client.Airtable, discord *client.Discord,
	poller *database.Poller, discovery *discovery.Poller, syncer *syncer.Syncer, state *state.State) {
	var bot = huntbotBot{
		ctx:       ctx,
		airtable:  airtable,
		discord:   discord,
		poller:    poller,
		discovery: discovery,
		syncer:    syncer,
		state:     state,
	}
	discord.AddCommand(bot.makeSlashCommand())
}

type huntbotBot struct {
	ctx       context.Context
	airtable  *client.Airtable
	discord   *client.Discord
	poller    *database.Poller
	discovery *discovery.Poller
	syncer    *syncer.Syncer
	state     *state.State
}

func (bot *huntbotBot) makeSlashCommand() *client.DiscordCommand {
	return &client.DiscordCommand{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "huntbot",
			Description: "Robot control panel 🤖",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "kill",
					Description: "Temporarily disable Huntbot ✋",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "start",
					Description: "Re-enable Huntbot 📡",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "nodiscovery",
					Description: "Stop Huntbot from discovering new puzzles 🔎",
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
		},
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			if i.IC.ChannelID != bot.discord.QMChannel.ID {
				return fmt.Sprintf(":tv: Please use `/huntbot` commands in the %s channel...",
					bot.discord.QMChannel.Mention()), nil
			}

			bot.state.Lock()
			defer bot.state.CommitAndUnlock()

			var reply string
			switch i.Subcommand.Name {
			case "kill":
				bot.state.DiscoveryDisabled = true
				if !bot.state.HuntbotDisabled {
					bot.state.HuntbotDisabled = true
					reply = "Ok, I've disabled the bot for now.  Enable it with `/huntbot start`."
				} else {
					reply = "The bot was already disabled. Enable it with `/huntbot start`."
				}
				bot.discord.UpdateStatus(bot.state) // best-effort, ignore errors
				return reply, nil
			case "start":
				bot.state.DiscoveryDisabled = false
				if bot.state.HuntbotDisabled {
					bot.state.HuntbotDisabled = false
					reply = "Ok, I've enabled the bot for now. Disable it with `/huntbot kill`."
				} else {
					reply = "The bot was already enabled. Disable it with `/huntbot kill`."
				}
				bot.discord.UpdateStatus(bot.state) // best-effort, ignore errors
				return reply, nil
			case "nodiscovery":
				if bot.discovery == nil {
					return "Huntbot is running without puzzle auto-discovery configured.", nil
				}
				bot.state.DiscoveryDisabled = true
				bot.discord.UpdateStatus(bot.state) // best-effort, ignore errors
				return "Ok, I've paused puzzle auto-discovery for now. Re-enable it with `/huntbot start`. " +
					"(This will also reenable the entire bot if the bot has been killed.)", nil
			case "yikes":
				if confirmOpt, err := bot.discord.OptionByName(i.Subcommand.Options, "confirm"); err != nil {
					return "", err
				} else if value := confirmOpt.StringValue(); value != "⚠️" {
					return fmt.Sprintf(":no_smoking: Incorrect confirmation value %q, operation aborted.",
						value), nil
				}
				go bot.fullResync(s, i)
				return ":warning: Initiated full re-sync!", nil
			default:
				return "", fmt.Errorf("unexpected /huntbot subcommand: %q", i.Subcommand.Name)
			}
		},
	}
}

func (bot *huntbotBot) fullResync(s *discordgo.Session, i *client.DiscordCommandInput) {
	var errs = make(map[string]error)

	puzzles, err := bot.airtable.ListPuzzles()
	if err == nil {
		for j, id := range puzzles {
			var puzzle *schema.Puzzle
			puzzle, err = bot.airtable.LockByID(id)
			if err != nil {
				err = fmt.Errorf("failed to load %q: %s", id, spew.Sdump(err))
				break
			}

			_, err = bot.syncer.ForceUpdate(bot.ctx, puzzle)
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
					err = fmt.Errorf("huntbot yikes: failed to update with progress: %v", err)
					break
				}
			}
		}
	}

	var msg string
	if err != nil {
		log.Printf("huntbot yikes: failed to re-sync: %v", err)
		msg = fmt.Sprintf(":warning: Full re-sync failed: ```\n%s\n```", spew.Sdump(err))
	} else if len(errs) > 0 {
		msg = ":warning: Full re-sync succeeded with errors:\n"
		for name, err := range errs {
			msg += fmt.Sprintf("%s\n```%s\n```\n", name, spew.Sdump(err))
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
