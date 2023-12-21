package bot

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"golang.org/x/xerrors"
)

type QMBot struct {
	discord *discord.Client
}

func NewQMBot(discord *discord.Client) discord.Bot {
	return &QMBot{discord}
}

func (b *QMBot) Register() (*discordgo.ApplicationCommand, bool) {
	return &discordgo.ApplicationCommand{
		Name:        "qm",
		Description: "Tools for the Quartermaster 👷",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "start",
				Description: "Start your shift as Quartermaster ☕",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
			{
				Name:        "stop",
				Description: "End your shift as Quartermaster 🛌",
				Type:        discordgo.ApplicationCommandOptionSubCommand,
			},
		},
	}, false
}

func (b *QMBot) Handle(ctx context.Context, input *discord.CommandInput) (string, error) {
	if input.IC.ChannelID != b.discord.QMChannel.ID {
		return fmt.Sprintf(":tv: Please use `/qm` commands in the %s channel...",
			b.discord.QMChannel.Mention()), nil
	}

	switch input.Subcommand.Name {
	case "start":
		if err := b.discord.MakeQM(input.User); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s is now a QM", input.User.Mention()), nil
	case "stop":
		if err := b.discord.UnMakeQM(input.User); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s is no longer a QM", input.User.Mention()), nil
	default:
		return "", xerrors.Errorf("unexpected /qm subcommand: %q", input.Subcommand.Name)
	}
}

func (b *QMBot) HandleScheduledEvent(context.Context,
	*discordgo.GuildScheduledEventUpdate) error {
	return nil
}
