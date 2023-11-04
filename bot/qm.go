package bot

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
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

func (b *QMBot) Handle(ctx context.Context, s *discordgo.Session,
	i *discord.CommandInput) (string, error) {

	if i.IC.ChannelID != b.discord.QMChannel.ID {
		return fmt.Sprintf(":tv: Please use `/qm` commands in the %s channel...",
			b.discord.QMChannel.Mention()), nil
	}

	switch i.Subcommand.Name {
	case "start":
		err := s.GuildMemberRoleAdd(b.discord.Guild.ID, i.User.ID, b.discord.QMRole.ID)
		if err != nil {
			return "", fmt.Errorf("unable to make %s a QM: %v", i.User.Mention(), err)
		}
		return fmt.Sprintf("%s is now a QM", i.User.Mention()), nil
	case "stop":
		err := s.GuildMemberRoleRemove(b.discord.Guild.ID, i.User.ID, b.discord.QMRole.ID)
		if err != nil {
			return "", fmt.Errorf("unable to remove %s from QM role: %v", i.User.Mention(), err)
		}
		return fmt.Sprintf("%s is no longer a QM", i.User.Mention()), nil
	default:
		return "", fmt.Errorf("unexpected /qm subcommand: %q", i.Subcommand.Name)
	}
}
