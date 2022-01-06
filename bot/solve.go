package bot

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/gauravjsingh/emojihunt/syncer"
)

func MakeSolveCommand(ctx context.Context, air *client.Airtable, dis *client.Discord, syn *syncer.Syncer) *client.DiscordCommand {
	return &client.DiscordCommand{
		InteractionType: discordgo.InteractionApplicationCommand,
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "solve",
			Description: "Use in a puzzle channel to mark the puzzle as solved! 🌠",
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
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			var newStatus schema.Status
			var err error
			var found = false
			for _, opt := range i.IC.ApplicationCommandData().Options {
				if opt.Name == "as" {
					if newStatus, err = schema.ParseTextStatus(opt.StringValue()); err != nil {
						return "", err
					}
					found = true
				}
			}
			if !found {
				return "", fmt.Errorf("could not find status argument in options list")
			}

			var answer string
			for _, opt := range i.IC.ApplicationCommandData().Options {
				if opt.Name == "answer" {
					answer = opt.StringValue()
				}
			}
			if answer == "" {
				return "", fmt.Errorf("could not find answer argument in options list")
			}

			puzzle, err := air.FindByDiscordChannel(i.IC.ChannelID)
			if err != nil {
				return ":butterfly: I can't find a puzzle associated with this channel. Is this a puzzle channel?", nil
			}

			return dis.ReplyAsync(s, i, func() (string, error) {
				if puzzle, err = air.MarkSolved(puzzle, newStatus, answer); err != nil {
					return "", err
				}
				if puzzle, err = syn.IdempotentCreateUpdate(ctx, puzzle, true); err != nil {
					return "", err
				}
				return fmt.Sprintf(
					"🎉 Congratulations on the %s! I'll record the answer `%s` and archive this channel.",
					newStatus.SolvedNoun(), answer,
				), nil
			})
		},
	}
}
