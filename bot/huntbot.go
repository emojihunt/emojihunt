package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/database"
	"github.com/gauravjsingh/emojihunt/discovery"
	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/gauravjsingh/emojihunt/state"
	"github.com/gauravjsingh/emojihunt/syncer"
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
		InteractionType: discordgo.InteractionApplicationCommand,
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
				{
					Name:        "error",
					Description: "Test error message 💥",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			if i.IC.ChannelID != bot.discord.TechChannel.ID {
				return fmt.Sprintf(":tv: Please use `/huntbot` commands in the %s channel...",
					bot.discord.TechChannel.Mention()), nil
			}

			bot.state.Lock()
			defer bot.state.CommitAndUnlock()

			switch i.Subcommand.Name {
			case "kill":
				bot.state.DiscoveryDisabled = true
				if !bot.state.HuntbotDisabled {
					bot.state.HuntbotDisabled = true
					bot.discord.ChannelSend(bot.discord.TechChannel,
						fmt.Sprintf("**bot disabled by %v**", i.User.Mention()))
					return "Ok, I've disabled the bot for now.  Enable it with `/huntbot start`.", nil
				} else {
					return "The bot was already disabled. Enable it with `/huntbot start`.", nil
				}
			case "start":
				bot.state.DiscoveryDisabled = false
				if bot.state.HuntbotDisabled {
					bot.state.HuntbotDisabled = false
					bot.discord.ChannelSend(bot.discord.TechChannel,
						fmt.Sprintf("**bot enabled by %v**", i.User.Mention()))
					return "Ok, I've enabled the bot for now. Disable it with `/huntbot kill`.", nil
				} else {
					return "The bot was already enabled. Disable it with `/huntbot kill`.", nil
				}
			case "nodiscovery":
				if bot.discovery == nil {
					return "Huntbot is running without puzzle auto-discovery configured.", nil
				}
				bot.state.DiscoveryDisabled = true
				bot.discord.ChannelSend(bot.discord.TechChannel,
					fmt.Sprintf("**discovery paused by %v**", i.User.Mention()))
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

	puzzles, err := bot.airtable.ListApprovedPuzzles()
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
				_, err = s.InteractionResponseEdit(
					s.State.User.ID, i.IC.Interaction, &discordgo.WebhookEdit{
						Content: fmt.Sprintf(
							":warning: Initiated full re-sync! (%d / %d)", j, len(puzzles),
						),
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
		s.State.User.ID, i.IC.Interaction, &discordgo.WebhookEdit{
			Content: msg,
		},
	)
	if err != nil {
		log.Printf("huntbot yikes: failed to update with status %q: %v", msg, err)
	}
}
