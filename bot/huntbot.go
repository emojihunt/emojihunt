package bot

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/database"
	"github.com/gauravjsingh/emojihunt/discovery"
	"github.com/gauravjsingh/emojihunt/state"
)

func RegisterHuntbotCommand(discord *client.Discord, poller *database.Poller, discovery *discovery.Poller, state *state.State) {
	var bot = huntbotBot{discord, poller, discovery, state}
	discord.AddCommand(bot.makeSlashCommand())
}

type huntbotBot struct {
	discord   *client.Discord
	poller    *database.Poller
	discovery *discovery.Poller
	state     *state.State
}

func (bot *huntbotBot) makeSlashCommand() *client.DiscordCommand {
	return &client.DiscordCommand{
		InteractionType: discordgo.InteractionApplicationCommand,
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "huntbot",
			Description: "Robot control panel 🤖",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "kill",
					Description: "Temporarily disable Huntbot ✋",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "start",
					Description: "Re-enable Huntbot 📡",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "nodiscovery",
					Description: "Stop Huntbot from discovering new puzzles 🔎",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:         "error",
					Description:  "Test error message 💥",
					Type:         discordgo.ApplicationCommandOptionSubCommand,
					Autocomplete: false,
				},
			},
		},
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			bot.state.Lock()
			defer bot.state.CommitAndUnlock()

			switch i.Subcommand.Name {
			case "kill":
				bot.state.DiscoveryDisabled = true
				if !bot.state.HuntbotDisabled {
					bot.state.HuntbotDisabled = true
					bot.discord.ChannelSend(bot.discord.TechChannel,
						fmt.Sprintf("**bot disabled by %v**", i.User.Mention()))
					return "Ok, I've disabled the bot for now.  Enable it with `/huntbot start`.", nil
				} else {
					return "The bot was already disabled. Enable it with `/huntbot start`.", nil
				}
			case "start":
				bot.state.DiscoveryDisabled = false
				if bot.state.HuntbotDisabled {
					bot.state.HuntbotDisabled = false
					bot.discord.ChannelSend(bot.discord.TechChannel,
						fmt.Sprintf("**bot enabled by %v**", i.User.Mention()))
					return "Ok, I've enabled the bot for now. Disable it with `/huntbot kill`.", nil
				} else {
					return "The bot was already enabled. Disable it with `/huntbot kill`.", nil
				}
			case "nodiscovery":
				if bot.discovery == nil {
					return "Huntbot is running without puzzle auto-discovery configured.", nil
				}
				bot.state.DiscoveryDisabled = true
				bot.discord.ChannelSend(bot.discord.TechChannel,
					fmt.Sprintf("**discovery paused by %v**", i.User.Mention()))
				return "Ok, I've paused puzzle auto-discovery for now. Re-enable it with `/huntbot start`. " +
					"(This will also reenable the entire bot if the bot has been killed.)", nil
			default:
				return "", fmt.Errorf("unexpected /huntbot subcommand: %q", i.Subcommand.Name)
			}
		},
	}
}
