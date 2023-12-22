package sync

import (
	"context"

	"github.com/bwmarrin/discordgo"
)

func (c *Client) UpdateBotStatus(ctx context.Context) error {
	var data discordgo.UpdateStatusData
	if !c.Discovery {
		data.Status = "idle"
	} else if c.state.IsEnabled(ctx) {
		data.Status = "online"
	} else {
		data.Status = "dnd"
		data.Activities = []*discordgo.Activity{
			{
				Name:  "Huntbot",
				Type:  discordgo.ActivityTypeCustom,
				State: "puzzle discovery paused",
			},
		}
	}
	return c.discord.UpdateStatus(data)
}
