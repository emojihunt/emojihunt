package bot

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/gauravjsingh/emojihunt/syncer"
)

func MakeStatusCommand(ctx context.Context, air *client.Airtable, dis *client.Discord, syn *syncer.Syncer) *client.DiscordCommand {
	return &client.DiscordCommand{
		InteractionType: discordgo.InteractionApplicationCommand,
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "status",
			Description: "Use in a puzzle channel to update the puzzle's status 🚥",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "to",
					Description: "What's the new status?",
					Required:    true,
					Type:        discordgo.ApplicationCommandOptionString,
					Choices: []*discordgo.ApplicationCommandOptionChoice{
						{Name: "Not Started", Value: "Not Started"}, // value is displayed in the mousover UI
						{Name: "✍️ Working", Value: schema.Working},
						{Name: "🗑️ Abandoned", Value: schema.Abandoned},
						// for the solved statuses, use /solve
					},
				},
			},
		},
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			var newStatus schema.Status
			var err error
			var found = false
			for _, opt := range i.IC.ApplicationCommandData().Options {
				if opt.Name == "to" {
					if newStatus, err = schema.ParseTextStatus(opt.StringValue()); err != nil {
						return "", err
					}
					found = true
				}
			}
			if !found {
				return "", fmt.Errorf("could not find status argument in options list")
			}

			puzzle, err := air.FindByDiscordChannel(i.IC.ChannelID)
			if err != nil {
				return ":butterfly: I can't find a puzzle associated with this channel. Is this a puzzle channel?", nil
			}

			if puzzle.Status == newStatus {
				return fmt.Sprintf(":elephant: This puzzle already has status %s", newStatus.Pretty()), nil
			}

			return dis.ReplyAsync(s, i, func() (string, error) {
				reply := fmt.Sprintf(":face_with_monocle: Updated puzzle status to %s!", newStatus.Pretty())
				if puzzle.Answer != "" {
					reply = fmt.Sprintf(":woozy_face: Updated puzzle status to %s and cleared answer `%s`. "+
						"Was that right?", newStatus.Pretty(), puzzle.Answer)
				}

				if puzzle, err = air.UpdateStatusAndClearAnswer(puzzle, newStatus); err != nil {
					return "", err
				}
				if puzzle, err = syn.IdempotentCreateUpdate(ctx, puzzle, false); err != nil {
					return "", err
				}
				return reply, nil
			})
		},
	}
}
