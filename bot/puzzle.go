package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/state/status"
	"golang.org/x/xerrors"
)

type PuzzleBot struct {
	discord *discord.Client
	state   *state.Client
}

func NewPuzzleBot(discord *discord.Client, state *state.Client) discord.Bot {
	return &PuzzleBot{discord, state}
}

func (b *PuzzleBot) Register() (*discordgo.ApplicationCommand, bool) {
	return &discordgo.ApplicationCommand{
		Name:        "puzzle",
		Description: "Use in a puzzle channel to update puzzle information 🧩",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "voice",
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Description: "Assign this puzzle to a voice room 📻",
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
			{
				Name:        "progress",
				Description: "Use in a puzzle channel when you start or stop work 🚧",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "to",
						Description: "What's the new status?",
						Required:    true,
						Type:        discordgo.ApplicationCommandOptionString,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							// Values are displayed in the mouseover UI, FYI, so don't use "" for NotStarted
							{Name: status.NotStarted.Pretty(), Value: status.AlternateNotStarted},
							{Name: status.Working.Pretty(), Value: status.Working},
							{Name: status.Abandoned.Pretty(), Value: status.Abandoned},
						},
					},
				},
			},
			{
				Name:        "solved",
				Description: "Use in a puzzle channel to mark the puzzle as solved! 🌠",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "as",
						Description: "What kind of solve was it?",
						Required:    true,
						Type:        discordgo.ApplicationCommandOptionString,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{Name: status.Solved.Pretty(), Value: status.Solved},
							// Discord can't display the gender modifier on this emoji
							{Name: "🤦 Backsolved", Value: status.Backsolved},
							{Name: status.Purchased.Pretty(), Value: status.Purchased},
						},
					},
					{
						Name:        "answer",
						Description: "What was the answer?",
						Required:    true,
						Type:        discordgo.ApplicationCommandOptionString,
					},
				},
			},
			{
				Name:        "note",
				Description: "Use in a puzzle channel to add or update the note 💵",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "set",
						Description: "What's the puzzle about?",
						Required:    false,
						Type:        discordgo.ApplicationCommandOptionString,
					},
				},
			},
			{
				Name:        "location",
				Description: "Use in a puzzle channel to add or update the location 📍",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "set",
						Description: "What should the location be set to?",
						Required:    false,
						Type:        discordgo.ApplicationCommandOptionString,
					},
				},
			},
		},
	}, true
}

func (b *PuzzleBot) Handle(ctx context.Context, input *discord.CommandInput) (string, error) {
	puzzle, err := b.state.GetPuzzleByChannel(ctx, input.IC.ChannelID)
	if errors.Is(err, sql.ErrNoRows) {
		return ":butterfly: I can't find a puzzle associated with this channel. Is this a puzzle channel?", nil
	} else if err != nil {
		return "", err
	}

	var reply string
	var newStatus status.Status
	var newAnswer string
	switch input.Subcommand {
	case "voice":
		var channel *discordgo.Channel
		if channelOpt, ok := input.Options["in"]; !ok {
			channel = b.discord.ChannelValue(channelOpt)
			reply = fmt.Sprintf("Set the room for puzzle %q to %s", puzzle.Name, channel.Mention())
		} else {
			reply = fmt.Sprintf("Removed the room for puzzle %q", puzzle.Name)
		}
		puzzle, err = b.state.UpdatePuzzle(ctx, puzzle.ID,
			func(puzzle *state.RawPuzzle) error {
				puzzle.VoiceRoom = channel.ID
				return nil
			},
		)
		if err != nil {
			return "", err
		}
	case "progress":
		if statusOpt, ok := input.Options["to"]; !ok {
			return "", xerrors.Errorf("missing option: to")
		} else if newStatus, err = status.ParseText(statusOpt.StringValue()); err != nil {
			return "", err
		}

		puzzle, err = b.state.UpdatePuzzle(ctx, puzzle.ID,
			func(puzzle *state.RawPuzzle) error {
				if puzzle.Status == newStatus {
					reply = fmt.Sprintf(":elephant: This puzzle already has status %s", newStatus.Pretty())
				} else if puzzle.Answer == "" {
					reply = fmt.Sprintf(":face_with_monocle: Updated puzzle status to %s!", newStatus.Pretty())
				} else {
					reply = fmt.Sprintf(":woozy_face: Updated puzzle status to %s and cleared answer `%s`. "+
						"Was that right?", newStatus.Pretty(), puzzle.Answer)
				}
				puzzle.Status = newStatus
				puzzle.Answer = ""
				return nil
			},
		)
		if err != nil {
			return "", err
		}
	case "solved":
		if statusOpt, ok := input.Options["as"]; !ok {
			return "", xerrors.Errorf("missing option: as")
		} else if newStatus, err = status.ParseText(statusOpt.StringValue()); err != nil {
			return "", err
		}

		if answerOpt, ok := input.Options["answer"]; !ok {
			return "", xerrors.Errorf("missing option: answer")
		} else {
			newAnswer = strings.ToUpper(answerOpt.StringValue())
		}

		_, err := b.state.UpdatePuzzle(ctx, puzzle.ID,
			func(puzzle *state.RawPuzzle) error {
				puzzle.Status = newStatus
				puzzle.Answer = newAnswer
				return nil
			},
		)
		if err != nil {
			return "", err
		}
		reply = fmt.Sprintf(
			"🎉 Congratulations on the %s! I'll record the answer `%s` and archive this channel.",
			newStatus.SolvedExclamation(), newAnswer,
		)
	case "note":
		var newNote string
		if noteOpt, ok := input.Options["set"]; ok {
			newNote = noteOpt.StringValue()
			reply = ":writing_hand: Updated puzzle note!"
		} else {
			reply = ":cl: Cleared puzzle note."
		}

		puzzle, err = b.state.UpdatePuzzle(ctx, puzzle.ID,
			func(puzzle *state.RawPuzzle) error {
				if puzzle.Note != "" {
					reply += fmt.Sprintf(" Previous note was: ```\n%s\n```", puzzle.Note)
				}
				puzzle.Note = newNote
				return nil
			},
		)
		if err != nil {
			return "", err
		}
	case "location":
		var newLocation string
		if locationOpt, ok := input.Options["set"]; ok {
			newLocation = locationOpt.StringValue()
			reply = ":writing_hand: Updated puzzle location!"
		} else {
			reply = ":cl: Cleared puzzle location."
		}

		_, err = b.state.UpdatePuzzle(ctx, puzzle.ID,
			func(puzzle *state.RawPuzzle) error {
				if puzzle.Location != "" {
					reply += fmt.Sprintf(" Previous location was: ```\n%s\n```", puzzle.Location)
				}
				puzzle.Location = newLocation
				return nil
			},
		)
		if err != nil {
			return "", err
		}
	default:
		return "", xerrors.Errorf("unexpected /puzzle subcommand: %q", input.Subcommand)
	}

	return reply, nil
}

func (b *PuzzleBot) HandleScheduledEvent(ctx context.Context,
	i *discordgo.GuildScheduledEventUpdate) error {

	if i.Status != discordgo.GuildScheduledEventStatusCompleted {
		return nil // ignore event
	}

	// We don't have to worry about double-processing puzzles because, even
	// though Discord *does* deliver events caused by the bot's own actions,
	// the bot uses *delete* to clean up events, while the Discord UI uses
	// an *update* to the "Completed" status. We only listen for the update
	// event, so we only see the human-triggered actions. (The bot does use
	// updates to update the name and to start the event initally, but
	// those events are filtered out by the condition above.)
	log.Printf("scheduled event %q was ended", i.Name)
	return b.state.ClearPuzzleVoiceRoom(ctx, i.ChannelID)
}
