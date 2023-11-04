package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/schema"
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
				Name:        "status",
				Description: "Use in a puzzle channel when you start or stop work 🚧",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "to",
						Description: "What's the new status?",
						Required:    true,
						Type:        discordgo.ApplicationCommandOptionString,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							// Values are displayed in the mouseover UI, so don't use "" for NotStarted
							{Name: "Not Started", Value: "Not Started"},
							{Name: "✍️ Working", Value: schema.Working},
							{Name: "🗑️ Abandoned", Value: schema.Abandoned},
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
							{Name: "🏅 Solved", Value: schema.Solved},
							{Name: "🤦 Backsolved", Value: schema.Backsolved},
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
				Name:        "description",
				Description: "Use in a puzzle channel to add or update the description 📝",
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

	var reply string
	var newStatus schema.Status
	var newAnswer string
	switch input.Subcommand.Name {
	case "status":
		if statusOpt, err := b.discord.OptionByName(input.Subcommand.Options, "to"); err != nil {
			return "", err
		} else if newStatus, err = schema.ParseTextStatus(statusOpt.StringValue()); err != nil {
			return "", err
		}

		if puzzle.Status == newStatus {
			return fmt.Sprintf(":elephant: This puzzle already has status %s", newStatus.Human()), nil
		}

		if puzzle.Answer == "" {
			reply = fmt.Sprintf(":face_with_monocle: Updated puzzle status to %s!", newStatus.Human())
		} else {
			reply = fmt.Sprintf(":woozy_face: Updated puzzle status to %s and cleared answer `%s`. "+
				"Was that right?", newStatus.Human(), puzzle.Answer)
		}

		if puzzle, err = b.db.SetStatusAndAnswer(ctx, puzzle, newStatus, newAnswer); err != nil {
			return "", err
		}
		if puzzle, err = b.syncer.HandleStatusChange(ctx, puzzle, true); err != nil {
			return "", err
		}
	case "solved":
		if statusOpt, err := b.discord.OptionByName(input.Subcommand.Options, "as"); err != nil {
			return "", err
		} else if newStatus, err = schema.ParseTextStatus(statusOpt.StringValue()); err != nil {
			return "", err
		}

		if answerOpt, err := b.discord.OptionByName(input.Subcommand.Options, "answer"); err != nil {
			return "", err
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
		if puzzle, err = b.syncer.HandleStatusChange(ctx, puzzle, true); err != nil {
			return "", err
		}
	case "description":
		var newDescription string
		if descriptionOpt, err := b.discord.OptionByName(input.Subcommand.Options, "is"); err == nil {
			newDescription = descriptionOpt.StringValue()
			reply = ":writing_hand: Updated puzzle description!"
		} else {
			reply = ":cl: Cleared puzzle description."
		}
		if puzzle.Description != "" {
			reply += fmt.Sprintf(" Previous description was: ```\n%s\n```", puzzle.Description)
		}

		if puzzle, err = b.db.SetDescription(ctx, puzzle, newDescription); err != nil {
			return "", err
		}
		if err = b.syncer.DiscordCreateUpdatePin(puzzle); err != nil {
			return "", err
		}
	case "location":
		var newLocation string
		if locationOpt, err := b.discord.OptionByName(input.Subcommand.Options, "is"); err == nil {
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
