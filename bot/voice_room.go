package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/schema"
	"github.com/emojihunt/emojihunt/syncer"
	"golang.org/x/xerrors"
)

type VoiceRoomBot struct {
	main    context.Context
	db      *db.Client
	discord *discord.Client
	syncer  *syncer.Syncer
}

func NewVoiceRoomBot(main context.Context, db *db.Client, discord *discord.Client,
	syncer *syncer.Syncer) discord.Bot {

	b := &VoiceRoomBot{main, db, discord, syncer}
	return b
}

func (b *VoiceRoomBot) Register() (*discordgo.ApplicationCommand, bool) {
	return &discordgo.ApplicationCommand{
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
	}, true
}

func (b *VoiceRoomBot) Handle(ctx context.Context, input *discord.CommandInput) (string, error) {
	puzzle, err := b.db.LockByDiscordChannel(ctx, input.IC.ChannelID)
	if err != nil {
		return "", err
	} else if puzzle == nil {
		return ":butterfly: I can't find a puzzle associated with this channel. Is this a puzzle channel?", nil
	}
	defer puzzle.Unlock()

	if problems := puzzle.Problems(); len(problems) > 0 {
		return fmt.Sprintf(":cold_sweat: I can't update this puzzle because it has errors in "+
			"Airtable. Please check %s for more information...", b.discord.QMChannel.Mention()), nil
	}

	b.syncer.VoiceRoomMutex.Lock()
	defer b.syncer.VoiceRoomMutex.Unlock()

	var reply string
	var channel *discordgo.Channel
	switch input.Subcommand.Name {
	case "start":
		channelOpt, err := b.discord.OptionByName(input.Subcommand.Options, "in")
		if err != nil {
			return "", err
		}
		channel = b.discord.ChannelValue(channelOpt)
		reply = fmt.Sprintf("Set the room for puzzle %q to %s", puzzle.Name, channel.Mention())
	case "stop":
		reply = fmt.Sprintf("Removed the room for puzzle %q", puzzle.Name)
	default:
		return "", xerrors.Errorf("unexpected /voice subcommand: %q", input.Subcommand.Name)
	}

	// Sync the change!
	if puzzle, err = b.db.SetVoiceRoom(ctx, puzzle, channel); err != nil {
		return "", err
	}
	if err = b.syncer.DiscordCreateUpdatePin(puzzle); err != nil {
		return "", err
	}
	if err = b.syncer.SyncVoiceRooms(ctx); err != nil {
		return "", err
	}

	return reply, nil
}

func (b *VoiceRoomBot) HandleScheduledEvent(
	ctx context.Context, i *discordgo.GuildScheduledEventUpdate) error {

	if i.Description != syncer.VoiceRoomEventDescription ||
		i.Status != discordgo.GuildScheduledEventStatusCompleted {
		return nil // ignore event
	}

	// We don't have to worry about double-processing puzzles because, even
	// though Discord *does* deliver events caused by the bot's own actions,
	// the bot uses *delete* to clean up events, while the Discord UI uses
	// an *update* to the "Completed" status. We only listen for the update
	// event, so we only see the human-triggered actions. (The bot does use
	// updates to update the name and to start the event initally, but
	// those events are filtered out by the condition above.)
	log.Printf("discord: processing scheduled event completion for %q", i.Name)

	b.syncer.VoiceRoomMutex.Lock()
	puzzles, err := b.db.ListWithVoiceRoom(ctx)
	b.syncer.VoiceRoomMutex.Unlock()
	if err != nil {
		return err
	}

	var final error
	for _, info := range puzzles {
		if err = b.clearVoiceRoom(ctx, &info, i.ChannelID); err != nil {
			final = err
		}
	}
	return final
}

func (b *VoiceRoomBot) clearVoiceRoom(ctx context.Context, info *schema.VoicePuzzle,
	expectedVoiceRoom string) error {

	puzzle, err := b.db.LockByID(ctx, info.ID)
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

	puzzle, err = b.db.SetVoiceRoom(ctx, puzzle, nil)
	if err != nil {
		return err
	}
	return b.syncer.DiscordCreateUpdatePin(puzzle)
}
