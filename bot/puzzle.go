package bot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/db/field"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
	"golang.org/x/xerrors"
)

type PuzzleBot struct {
	discord *discord.Client
	state   *state.Client
	syncer  *syncer.Syncer
}

func NewPuzzleBot(discord *discord.Client, state *state.Client, syncer *syncer.Syncer) discord.Bot {
	return &PuzzleBot{discord, state, syncer}
}

func (b *PuzzleBot) Register() (*discordgo.ApplicationCommand, bool) {
	return &discordgo.ApplicationCommand{
		Name:        "puzzle",
		Description: "Use in a puzzle channel to update puzzle information 🧩",
		Options: []*discordgo.ApplicationCommandOption{
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
							{Name: field.StatusNotStarted.Pretty(), Value: field.AlternateNotStarted},
							{Name: field.StatusWorking.Pretty(), Value: field.StatusWorking},
							{Name: field.StatusAbandoned.Pretty(), Value: field.StatusAbandoned},
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
							{Name: field.StatusSolved.Pretty(), Value: field.StatusSolved},
							// Discord can't display the gender modifier on this emoji
							{Name: "🤦 Backsolved", Value: field.StatusBacksolved},
							{Name: field.StatusPurchased.Pretty(), Value: field.StatusPurchased},
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
						Name:        "is",
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
						Name:        "is",
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
	var newStatus field.Status
	var newAnswer string
	switch input.Subcommand.Name {
	case "progress":
		if statusOpt, ok := b.discord.OptionByName(input.Subcommand.Options, "to"); !ok {
			return "", xerrors.Errorf("missing option: to")
		} else if newStatus, err = field.ParseTextStatus(statusOpt.StringValue()); err != nil {
			return "", err
		}

		puzzle, err = b.state.UpdatePuzzle(ctx, puzzle.ID,
			func(puzzle *db.RawPuzzle) error {
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
		_, err = b.syncer.HandleStatusChange(ctx, puzzle, true)
		if err != nil {
			return "", err
		}
	case "solved":
		if statusOpt, ok := b.discord.OptionByName(input.Subcommand.Options, "as"); !ok {
			return "", xerrors.Errorf("missing option: as")
		} else if newStatus, err = field.ParseTextStatus(statusOpt.StringValue()); err != nil {
			return "", err
		}

		if answerOpt, ok := b.discord.OptionByName(input.Subcommand.Options, "answer"); !ok {
			return "", xerrors.Errorf("missing option: answer")
		} else {
			newAnswer = strings.ToUpper(answerOpt.StringValue())
		}

		puzzle, err := b.state.UpdatePuzzle(ctx, puzzle.ID,
			func(puzzle *db.RawPuzzle) error {
				puzzle.Status = newStatus
				puzzle.Answer = newAnswer
				return nil
			},
		)
		if err != nil {
			return "", err
		}
		if _, err = b.syncer.HandleStatusChange(ctx, puzzle, true); err != nil {
			return "", err
		}
		reply = fmt.Sprintf(
			"🎉 Congratulations on the %s! I'll record the answer `%s` and archive this channel.",
			newStatus.SolvedExclamation(), newAnswer,
		)
	case "note":
		var newNote string
		if noteOpt, ok := b.discord.OptionByName(input.Subcommand.Options, "is"); ok {
			newNote = noteOpt.StringValue()
			reply = ":writing_hand: Updated puzzle note!"
		} else {
			reply = ":cl: Cleared puzzle note."
		}

		puzzle, err = b.state.UpdatePuzzle(ctx, puzzle.ID,
			func(puzzle *db.RawPuzzle) error {
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
		if err = b.syncer.DiscordCreateUpdatePin(puzzle); err != nil {
			return "", err
		}
	case "location":
		var newLocation string
		if locationOpt, ok := b.discord.OptionByName(input.Subcommand.Options, "is"); ok {
			newLocation = locationOpt.StringValue()
			reply = ":writing_hand: Updated puzzle location!"
		} else {
			reply = ":cl: Cleared puzzle location."
		}

		puzzle, err = b.state.UpdatePuzzle(ctx, puzzle.ID,
			func(puzzle *db.RawPuzzle) error {
				if puzzle.Location != "" {
					reply += fmt.Sprintf(" Previous location was: ```\n%s\n```", puzzle.Location)
				}
				puzzle.Location = newLocation
				return nil
			},
		)
		if err = b.syncer.DiscordCreateUpdatePin(puzzle); err != nil {
			return "", err
		}
	default:
		return "", xerrors.Errorf("unexpected /puzzle subcommand: %q", input.Subcommand.Name)
	}

	return reply, nil
}

func (b *PuzzleBot) HandleScheduledEvent(context.Context,
	*discordgo.GuildScheduledEventUpdate) error {
	return nil
}
