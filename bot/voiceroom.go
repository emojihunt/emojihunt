package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
)

const roomStatusHeader = "Working Room"

func MakeVoiceRoomCommand(air *client.Airtable, dis *client.Discord) *client.DiscordCommand {
	return &client.DiscordCommand{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "room",
			Description: "Assign puzzles to voice rooms",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Assign this puzzle to a voice room 🔔",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "in",
							Description: "Where are we going? 🗺️",
							Required:    true,
							Type:        discordgo.ApplicationCommandOptionChannel,
							ChannelTypes: []discordgo.ChannelType{
								discordgo.ChannelTypeGuildVoice,
							},
						},
					},
				},
				{
					Name:        "stop",
					Description: "Remove this puzzle from its voice room 🔕",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			puzzle, err := air.FindByDiscordChannel(i.IC.ChannelID)
			if err != nil {
				return "", fmt.Errorf("unable to get puzzle for channel ID %q", i.IC.ChannelID)
			}

			switch i.Subcommand.Name {
			case "start":
				var channel *discordgo.Channel
				for _, opt := range i.Subcommand.Options {
					if opt.Name == "in" {
						channel = opt.ChannelValue(s)
					}
				}
				if channel == nil {
					return "", fmt.Errorf("could not find channel argument in options list")
				}
				setPinnedVoiceInfo(dis, i.IC.ChannelID, channel)
				return fmt.Sprintf("Set the room for puzzle %q to %s", puzzle.Name, channel.Mention()), nil
			case "stop":
				setPinnedVoiceInfo(dis, i.IC.ChannelID, nil)
				return fmt.Sprintf("Removed the room for puzzle %q", puzzle.Name), nil
			default:
				return "", fmt.Errorf("unexpected /room subcommand: %q", i.Subcommand.Name)
			}
		},
	}
}

func setPinnedVoiceInfo(dis *client.Discord, puzzleChannelID string, voiceChannel *discordgo.Channel) error {
	room := "No voice room set. Use `/room start $room` to start working in $room."
	if voiceChannel != nil {
		room = fmt.Sprintf("Join us in %s!", voiceChannel.Mention())
	}
	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: roomStatusHeader},
		Description: room,
	}
	return dis.CreateUpdatePin(puzzleChannelID, roomStatusHeader, embed)
}
