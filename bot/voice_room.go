package bot

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/schema"
	"github.com/emojihunt/emojihunt/syncer"
)

type VoiceRoomBot struct {
	db      *db.Client
	discord *discord.Client
	syncer  *syncer.Syncer
}

func NewVoiceRoomBot(db *db.Client, discord *discord.Client, syncer *syncer.Syncer) discord.Bot {
	b := &VoiceRoomBot{db, discord, syncer}
	discord.AddHandler(b.scheduledEventUpdateHandler)
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

func (b *VoiceRoomBot) Handle(s *discordgo.Session, i *discord.CommandInput) (string, error) {
	puzzle, err := b.db.LockByDiscordChannel(i.IC.ChannelID)
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
	switch i.Subcommand.Name {
	case "start":
		channelOpt, err := b.discord.OptionByName(i.Subcommand.Options, "in")
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
	if puzzle, err = b.db.SetVoiceRoom(puzzle, channel); err != nil {
		return "", err
	}
	if err = b.syncer.DiscordCreateUpdatePin(puzzle); err != nil {
		return "", err
	}
	if err = b.syncer.SyncVoiceRooms(context.TODO()); err != nil {
		return "", err
	}

	return reply, nil
}

func (b *VoiceRoomBot) scheduledEventUpdateHandler(s *discordgo.Session, i *discordgo.GuildScheduledEventUpdate) {
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

	b.syncer.VoiceRoomMutex.Lock()
	puzzles, err := b.db.ListWithVoiceRoom()
	b.syncer.VoiceRoomMutex.Unlock()

	if err != nil {
		log.Printf("discord: error processing scheduled event completion: %v", spew.Sdump(err))
		return
	}

	var errs []error
	for _, info := range puzzles {
		if err = b.clearVoiceRoom(&info, i.ChannelID); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		log.Printf("discord: errors processing scheduled event completion: %v", spew.Sdump(errs))
	}
}

func (b *VoiceRoomBot) clearVoiceRoom(info *schema.VoicePuzzle, expectedVoiceRoom string) error {
	puzzle, err := b.db.LockByID(info.ID)
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

	puzzle, err = b.db.SetVoiceRoom(puzzle, nil)
	if err != nil {
		return err
	}
	return b.syncer.DiscordCreateUpdatePin(puzzle)
}
