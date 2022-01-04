package bot

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/client"
)

type hunt struct {
	start, end time.Time
}

var hunts = []hunt{
	{time.Date(2021, 1, 15, 17, 0, 0, 0, time.UTC), time.Date(2021, 1, 18, 15, 0, 0, 0, time.UTC)},
	{time.Date(2022, 1, 14, 17, 0, 0, 0, time.UTC), time.Date(2022, 1, 16, 23, 0, 0, 0, time.UTC)},
	{time.Date(2023, 1, 13, 17, 0, 0, 0, time.UTC), time.Date(2023, 1, 15, 23, 0, 0, 0, time.UTC)},
}

func MakeHuntYetCommand() *client.DiscordCommand {
	return &client.DiscordCommand{
		ApplicationCommand: &discordgo.ApplicationCommand{
			Name:        "/huntyet",
			Description: "IS IT HUNT YET???",
		},
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			var now = time.Now()
			for _, h := range hunts {
				if h.start.After(now) {
					return fmt.Sprintf("No. You'll have to wait another %v.", formatDuration(h.start.Sub(now))), nil
				} else if h.end.After(now) {
					return "Yes! HUNT HUNT HUNT!", nil
				}
			}
			return "Gosh, I'm not sure! @tech can update the bot.", nil
		},
	}
}

func formatDuration(d time.Duration) string {
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	millis := int(d.Milliseconds()) % 1000
	return fmt.Sprintf("%v days, %v hours, %v minutes, %v.%02d seconds", days, hours, minutes, seconds, millis)
}
