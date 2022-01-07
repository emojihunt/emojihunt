package bot

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/gauravjsingh/emojihunt/syncer"
)

func MakePuzzleCommand(ctx context.Context, air *client.Airtable, dis *client.Discord, syn *syncer.Syncer) *client.DiscordCommand {
	return &client.DiscordCommand{
		InteractionType: discordgo.InteractionApplicationCommand,
		ApplicationCommand: &discordgo.ApplicationCommand{
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
								{Name: "🤦‍♀️ Backsolved", Value: schema.Backsolved},
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
			},
		},
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			puzzle, err := air.FindByDiscordChannel(i.IC.ChannelID)
			if err != nil {
				return ":butterfly: I can't find a puzzle associated with this channel. Is this a puzzle channel?", nil
			}

			var reply string
			var newStatus schema.Status
			var newAnswer string
			switch i.Subcommand.Name {
			case "status":
				if statusOpt, err := dis.OptionByName(i.Subcommand.Options, "to"); err != nil {
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
			case "solved":
				if statusOpt, err := dis.OptionByName(i.Subcommand.Options, "as"); err != nil {
					return "", err
				} else if newStatus, err = schema.ParseTextStatus(statusOpt.StringValue()); err != nil {
					return "", err
				}

				if answerOpt, err := dis.OptionByName(i.Subcommand.Options, "answer"); err != nil {
					return "", err
				} else {
					newAnswer = strings.ToUpper(answerOpt.StringValue())
				}

				reply = fmt.Sprintf(
					"🎉 Congratulations on the %s! I'll record the answer `%s` and archive this channel.",
					newStatus.SolvedNoun(), newAnswer,
				)
			default:
				return "", fmt.Errorf("unexpected /puzzle subcommand: %q", i.Subcommand.Name)
			}

			return dis.ReplyAsync(s, i, func() (string, error) {
				if puzzle, err = air.SetStatusAndAnswer(puzzle, newStatus, newAnswer); err != nil {
					return "", err
				}
				if puzzle, err = syn.BasicUpdate(ctx, puzzle, true); err != nil {
					return "", err
				}
				return reply, nil
			})
		},
	}
}
