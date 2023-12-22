package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
	"golang.org/x/xerrors"
)

type VoiceRoomBot struct {
	main    context.Context
	discord *discord.Client
	state   *state.Client
	syncer  *syncer.Syncer
}

func NewVoiceRoomBot(main context.Context, discord *discord.Client,
	state *state.Client, syncer *syncer.Syncer) discord.Bot {

	b := &VoiceRoomBot{main, discord, state, syncer}
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
	puzzle, err := b.state.GetPuzzleByChannel(ctx, input.IC.ChannelID)
	if errors.Is(err, sql.ErrNoRows) {
		return ":butterfly: I can't find a puzzle associated with this channel. Is this a puzzle channel?", nil
	} else if err != nil {
		return "", err
	}

	b.syncer.VoiceRoomMutex.Lock()
	defer b.syncer.VoiceRoomMutex.Unlock()

	var reply string
	var channel *discordgo.Channel
	switch input.Subcommand.Name {
	case "start":
		channelOpt, ok := b.discord.OptionByName(input.Subcommand.Options, "in")
		if !ok {
			return "", xerrors.Errorf("missing option: in")
		}
		channel = b.discord.ChannelValue(channelOpt)
		reply = fmt.Sprintf("Set the room for puzzle %q to %s", puzzle.Name, channel.Mention())
	case "stop":
		reply = fmt.Sprintf("Removed the room for puzzle %q", puzzle.Name)
	default:
		return "", xerrors.Errorf("unexpected /voice subcommand: %q", input.Subcommand.Name)
	}

	// Sync the change!
	puzzle, err = b.state.UpdatePuzzle(ctx, puzzle.ID,
		func(puzzle *state.RawPuzzle) error {
			puzzle.VoiceRoom = channel.ID
			return nil
		},
	)
	if err != nil {
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
	defer b.syncer.VoiceRoomMutex.Unlock()

	return b.state.ClearPuzzleVoiceRoom(ctx, i.ChannelID)
}
