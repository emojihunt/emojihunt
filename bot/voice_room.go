package bot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/syncer"
)

type VoiceRoomBot struct {
	ctx      context.Context
	airtable *client.Airtable
	discord  *client.Discord
	syncer   *syncer.Syncer
}

func NewVoiceRoomBot(ctx context.Context, airtable *client.Airtable, discord *client.Discord, syncer *syncer.Syncer) *VoiceRoomBot {
	return &VoiceRoomBot{ctx, airtable, discord, syncer}
}

func (bot *VoiceRoomBot) MakeSlashCommand() *client.DiscordCommand {
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
			bot.syncer.VoiceRoomMutex.Lock()
			defer bot.syncer.VoiceRoomMutex.Unlock()

			puzzle, err := bot.airtable.FindByDiscordChannel(i.IC.ChannelID)
			if err != nil {
				return "", fmt.Errorf("unable to get puzzle for channel ID %q", i.IC.ChannelID)
			}

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

			// Search for an existing event for the given voice room
			events, err := bot.discord.ListScheduledEvents()
			if err != nil {
				return "", err
			}
			var event *discordgo.GuildScheduledEvent
			for _, item := range events {
				if channel != nil && item.ChannelID != nil && *item.ChannelID == channel.ID {
					event = item
				}
			}

			// If there's no existing event in this voice room, create one
			if event == nil && channel != nil {
				log.Printf("creating scheduled event in %s", channel.Name)
				start := time.Now().Add(5 * time.Minute)
				event, err = bot.discord.CreateScheduledEvent(&discordgo.GuildScheduledEvent{
					ChannelID:          &channel.ID,
					Name:               puzzle.Name,
					PrivacyLevel:       discordgo.GuildScheduledEventPrivacyLevelGuildOnly,
					ScheduledStartTime: &start,
					Description:        syncer.VoiceRoomEventDescription,
					EntityType:         discordgo.GuildScheduledEventEntityTypeVoice,
				})
				if err != nil {
					return "", err
				}

				event, err = bot.discord.UpdateScheduledEvent(event, map[string]interface{}{
					"status": discordgo.GuildScheduledEventStatusActive,
				})
				if err != nil {
					return "", err
				}
			}

			// Sync the change!
			if event == nil {
				puzzle, err = bot.airtable.UpdateVoiceRoomEvent(puzzle, "")
			} else {
				puzzle, err = bot.airtable.UpdateVoiceRoomEvent(puzzle, event.ID)
			}
			if err != nil {
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

func (bot *VoiceRoomBot) ScheduledEventUpdateHandler(s *discordgo.Session, i *discordgo.GuildScheduledEventUpdate) {
	if i.Description != syncer.VoiceRoomEventDescription || i.Status != discordgo.GuildScheduledEventStatusCompleted {
		return
	}

	bot.syncer.VoiceRoomMutex.Lock()
	defer bot.syncer.VoiceRoomMutex.Unlock()

	// We don't have to worry about double-processing puzzles because, even
	// though Discord *does* deliver events caused by the bot's own actions,
	// the bot uses *delete* to clean up events, while the Discord UI uses
	// an *update* to the "Completed" status. We only listen for the update
	// event, so we only see the human-triggered actions. (The bot does use
	// updates to update the name and to start the event initally, but
	// those events are filtered out by the condition above.)
	log.Printf("discord: processing scheduled event completion event for %q", i.Name)
	puzzles, err := bot.airtable.FindWithVoiceRoomEvent()
	if err == nil {
		for _, puzzle := range puzzles {
			puzzle, err = bot.airtable.UpdateVoiceRoomEvent(puzzle, "")
			if err != nil {
				break
			}
			if err = bot.syncer.DiscordCreateUpdatePin(puzzle); err != nil {
				break
			}
		}
	}
	if err != nil {
		log.Printf("discord: error processing scheduled event completion event: %v", spew.Sdump(err))
	}
}
