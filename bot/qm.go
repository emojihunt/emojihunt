package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
)

func RegisterQMBot(discord *discord.Client) {
	var bot = qmBot{discord}
	discord.AddCommand(bot.makeSlashCommand())
}

type qmBot struct {
	discord *discord.Client
}

func (bot *qmBot) makeSlashCommand() *discord.Command {
	return &discord.Command{
		ApplicationCommand: &discordgo.ApplicationCommand{
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
		},
		Handler: func(s *discordgo.Session, i *discord.CommandInput) (string, error) {
			if i.IC.ChannelID != bot.discord.QMChannel.ID {
				return fmt.Sprintf(":tv: Please use `/qm` commands in the %s channel...",
					bot.discord.QMChannel.Mention()), nil
			}

			switch i.Subcommand.Name {
			case "start":
				err := s.GuildMemberRoleAdd(bot.discord.Guild.ID, i.User.ID, bot.discord.QMRole.ID)
				if err != nil {
					return "", fmt.Errorf("unable to make %s a QM: %v", i.User.Mention(), err)
				}
				return fmt.Sprintf("%s is now a QM", i.User.Mention()), nil
			case "stop":
				err := s.GuildMemberRoleRemove(bot.discord.Guild.ID, i.User.ID, bot.discord.QMRole.ID)
				if err != nil {
					return "", fmt.Errorf("unable to remove %s from QM role: %v", i.User.Mention(), err)
				}
				return fmt.Sprintf("%s is no longer a QM", i.User.Mention()), nil
			default:
				return "", fmt.Errorf("unexpected /qm subcommand: %q", i.Subcommand.Name)
			}
		},
	}
}
