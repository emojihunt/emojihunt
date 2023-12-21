package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/db/field"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/syncer"
	"golang.org/x/xerrors"
)

type PuzzleBot struct {
	db      *db.Client
	discord *discord.Client
	syncer  *syncer.Syncer
}

func NewPuzzleBot(db *db.Client, discord *discord.Client, syncer *syncer.Syncer) discord.Bot {
	return &PuzzleBot{db, discord, syncer}
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
	puzzle, err := b.db.LoadByDiscordChannel(ctx, input.IC.ChannelID)
	if err != nil {
		return "", err
	} else if puzzle == nil {
		return ":butterfly: I can't find a puzzle associated with this channel. Is this a puzzle channel?", nil
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

		if puzzle.Status == newStatus {
			return fmt.Sprintf(":elephant: This puzzle already has status %s", newStatus.Pretty()), nil
		}

		if puzzle.Answer == "" {
			reply = fmt.Sprintf(":face_with_monocle: Updated puzzle status to %s!", newStatus.Pretty())
		} else {
			reply = fmt.Sprintf(":woozy_face: Updated puzzle status to %s and cleared answer `%s`. "+
				"Was that right?", newStatus.Pretty(), puzzle.Answer)
		}

		if puzzle, err = b.db.SetStatusAndAnswer(ctx, puzzle, newStatus, newAnswer); err != nil {
			return "", err
		}
		if _, err = b.syncer.HandleStatusChange(ctx, puzzle, true); err != nil {
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

		reply = fmt.Sprintf(
			"🎉 Congratulations on the %s! I'll record the answer `%s` and archive this channel.",
			newStatus.SolvedNoun(), newAnswer,
		)

		if puzzle, err = b.db.SetStatusAndAnswer(ctx, puzzle, newStatus, newAnswer); err != nil {
			return "", err
		}
		if _, err = b.syncer.HandleStatusChange(ctx, puzzle, true); err != nil {
			return "", err
		}
	case "note":
		var newNote string
		if noteOpt, ok := b.discord.OptionByName(input.Subcommand.Options, "is"); ok {
			newNote = noteOpt.StringValue()
			reply = ":writing_hand: Updated puzzle note!"
		} else {
			reply = ":cl: Cleared puzzle note."
		}
		if puzzle.Note != "" {
			reply += fmt.Sprintf(" Previous note was: ```\n%s\n```", puzzle.Note)
		}

		if puzzle, err = b.db.SetNote(ctx, puzzle, newNote); err != nil {
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
		if puzzle.Location != "" {
			reply += fmt.Sprintf(" Previous location was: ```\n%s\n```", puzzle.Location)
		}

		if puzzle, err = b.db.SetLocation(ctx, puzzle, newLocation); err != nil {
			return "", err
		}
		if err = b.syncer.DiscordCreateUpdatePin(puzzle); err != nil {
			return "", err
		}
	default:
		return "", xerrors.Errorf("unexpected /puzzle subcommand: %q", input.Subcommand.Name)
	}

	return reply, nil
}

func (b *PuzzleBot) HandleReaction(context.Context,
	*discordgo.MessageReaction) error {
	return nil
}

func (b *PuzzleBot) HandleScheduledEvent(context.Context,
	*discordgo.GuildScheduledEventUpdate) error {
	return nil
}
