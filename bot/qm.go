package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
)

func MakeQMCommand(dis *client.Discord) *client.DiscordCommand {
	return &client.DiscordCommand{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name: "qm",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Start your shift as Quartermaster :coffee:",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "stop",
					Description: "End your shift as Quartermaster :sleeping:",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			switch i.Subcommand {
			case "start":
				err := s.GuildMemberRoleAdd(dis.GuildID, i.User.ID, dis.QMRoleID)
				if err != nil {
					return "", fmt.Errorf("unable to make %s a QM: %v", i.User.Mention(), err)
				}
				return fmt.Sprintf("%s is now a QM", i.User.Mention()), nil
			case "stop":
				err := s.GuildMemberRoleRemove(dis.GuildID, i.User.ID, dis.QMRoleID)
				if err != nil {
					return "", fmt.Errorf("unable to remove %s from QM role: %v", i.User.Mention(), err)
				}
				return fmt.Sprintf("%s is no longer a QM", i.User.Mention()), nil
			default:
				return "", fmt.Errorf("unexpected /qm subcommand: %q", i.Subcommand)
			}
		},
	}
}
