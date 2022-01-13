package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/gauravjsingh/emojihunt/syncer"
)

func RegisterVoiceRoomBot(ctx context.Context, airtable *client.Airtable, discord *client.Discord, syncer *syncer.Syncer) {
	var bot = voiceRoomBot{ctx, airtable, discord, syncer}
	discord.AddCommand(bot.makeSlashCommand())
	discord.AddHandler(bot.scheduledEventUpdateHandler)
}

type voiceRoomBot struct {
	ctx      context.Context
	airtable *client.Airtable
	discord  *client.Discord
	syncer   *syncer.Syncer
}

func (bot *voiceRoomBot) makeSlashCommand() *client.DiscordCommand {
	return &client.DiscordCommand{
		InteractionType: discordgo.InteractionApplicationCommand,
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "voice",
			Description: "Assign puzzles to voice rooms",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Assign this puzzle to a voice room 🔔",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "in",
							Description: "Where are we going? 🗺️",
							Required:    true,
							Type:        discordgo.ApplicationCommandOptionChannel,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildVoice,
							},
						},
					},
				},
				{
					Name:        "stop",
					Description: "Remove this puzzle from its voice room 🔕",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Async: true,
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			puzzle, err := bot.airtable.LockByDiscordChannel(i.IC.ChannelID)
			if err != nil {
				return "", err
			} else if puzzle == nil {
				return ":butterfly: I can't find a puzzle associated with this channel. Is this a puzzle channel?", nil
			}
			defer puzzle.Unlock()

			bot.syncer.VoiceRoomMutex.Lock()
			defer bot.syncer.VoiceRoomMutex.Unlock()

			var reply string
			var channel *discordgo.Channel
			switch i.Subcommand.Name {
			case "start":
				channelOpt, err := bot.discord.OptionByName(i.Subcommand.Options, "in")
				if err != nil {
					return "", err
				}
				channel = channelOpt.ChannelValue(s)
				reply = fmt.Sprintf("Set the room for puzzle %q to %s", puzzle.Name, channel.Mention())
			case "stop":
				reply = fmt.Sprintf("Removed the room for puzzle %q", puzzle.Name)
			default:
				return "", fmt.Errorf("unexpected /voice subcommand: %q", i.Subcommand.Name)
			}

			// Sync the change!
			if puzzle, err = bot.airtable.SetVoiceRoom(puzzle, channel); err != nil {
				return "", err
			}
			if err = bot.syncer.DiscordCreateUpdatePin(puzzle); err != nil {
				return "", err
			}
			if err = bot.syncer.SyncVoiceRooms(bot.ctx); err != nil {
				return "", err
			}

			return reply, nil
		},
	}
}

func (bot *voiceRoomBot) scheduledEventUpdateHandler(s *discordgo.Session, i *discordgo.GuildScheduledEventUpdate) {
	if i.Description != syncer.VoiceRoomEventDescription || i.Status != discordgo.GuildScheduledEventStatusCompleted {
		return
	}

	// We don't have to worry about double-processing puzzles because, even
	// though Discord *does* deliver events caused by the bot's own actions,
	// the bot uses *delete* to clean up events, while the Discord UI uses
	// an *update* to the "Completed" status. We only listen for the update
	// event, so we only see the human-triggered actions. (The bot does use
	// updates to update the name and to start the event initally, but
	// those events are filtered out by the condition above.)
	log.Printf("discord: processing scheduled event completion for %q", i.Name)

	bot.syncer.VoiceRoomMutex.Lock()
	puzzles, err := bot.airtable.ListWithVoiceRoom()
	bot.syncer.VoiceRoomMutex.Unlock()

	if err != nil {
		log.Printf("discord: error processing scheduled event completion: %v", spew.Sdump(err))
		return
	}

	var errs []error
	for _, info := range puzzles {
		if err = bot.clearVoiceRoom(&info, *i.ChannelID); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		log.Printf("discord: errors processing scheduled event completion: %v", spew.Sdump(errs))
	}
}

func (bot *voiceRoomBot) clearVoiceRoom(info *schema.VoicePuzzle, expectedVoiceRoom string) error {
	puzzle, err := bot.airtable.LockByID(info.RecordID)
	if err != nil {
		return err
	}
	defer puzzle.Unlock()

	if puzzle.VoiceRoom != expectedVoiceRoom {
		// We've let go of VoiceRoomMutex (since we aren't allowed to
		// acquire the puzzle lock when holding it), so we need to
		// double-check that the puzzle hasn't changed.
		return nil
	}

	puzzle, err = bot.airtable.SetVoiceRoom(puzzle, nil)
	if err != nil {
		return err
	}
	return bot.syncer.DiscordCreateUpdatePin(puzzle)
}
