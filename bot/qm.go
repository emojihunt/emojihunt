package bot

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/sync"
	"golang.org/x/xerrors"
)

type QMBot struct {
	discord *discord.Client
	state   *state.Client
	sync    *sync.Client
}

func NewQMBot(discord *discord.Client, state *state.Client, sync *sync.Client) discord.Bot {
	return &QMBot{discord, state, sync}
}

func (b *QMBot) Register() (*discordgo.ApplicationCommand, bool) {
	return &discordgo.ApplicationCommand{
		Name:        "qm",
		Description: "Tools for the Quartermaster 👷",
		Options: []*discordgo.ApplicationCommandOption{
			{
				Name:        "shift",
				Description: "Start or end your shift as Quartermaster.",
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
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
			{
				Name:        "discovery",
				Description: "Pause or resume puzzle discovery.",
				Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
				Options: []*discordgo.ApplicationCommandOption{
					{
						Name:        "pause",
						Description: "Temporarily disable puzzle discovery ✋",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
					},
					{
						Name:        "resume",
						Description: "Re-enable puzzle discovery 📡",
						Type:        discordgo.ApplicationCommandOptionSubCommand,
					},
				},
			},
		},
	}, false
}

func (b *QMBot) Handle(ctx context.Context, input *discord.CommandInput) (string, error) {
	if input.IC.ChannelID != b.discord.QMChannel.ID {
		return fmt.Sprintf(":tv: Please use `/qm` commands in the %s channel...",
			b.discord.QMChannel.Mention()), nil
	}

	switch input.Subcommand {
	case "shift.start":
		if err := b.discord.MakeQM(input.User); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s is now a QM", input.User.Mention()), nil
	case "shift.stop":
		if err := b.discord.UnMakeQM(input.User); err != nil {
			return "", err
		}
		return fmt.Sprintf("%s is no longer a QM", input.User.Mention()), nil
	case "discovery.pause":
		if !b.sync.Discovery {
			return "Puzzle discovery isn't configured.", nil
		} else if b.state.EnableDiscovery(ctx, false) {
			return "Ok, I've paused puzzle discovery. Re-enable it with `/qm discovery resume`.",
				b.sync.UpdateBotStatus(ctx)
		} else {
			return "Discovery was already paused. Re-enable it with `/qm discovery resume`.", nil
		}
	case "discovery.resume":
		if !b.sync.Discovery {
			return "Puzzle discovery isn't configured.", nil
		} else if b.state.EnableDiscovery(ctx, true) {
			return "Ok, I've resumed puzzle discovery. Pause it with `/qm discovery pause`.",
				b.sync.UpdateBotStatus(ctx)
		} else {
			return "Discovery was already enabled. Pause it with `/qm discovery pause`.", nil
		}
	default:
		return "", xerrors.Errorf("unexpected /qm subcommand: %q", input.Subcommand)
	}
}

func (b *QMBot) HandleScheduledEvent(context.Context,
	*discordgo.GuildScheduledEventUpdate) error {
	return nil
}
