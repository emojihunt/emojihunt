package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/schema"
)

type HuntYetBot struct {
	duration time.Duration
	hunts    []time.Time
}

func NewHuntYetBot() discord.Bot {
	return &HuntYetBot{
		duration: 72 * time.Hour,
		hunts: []time.Time{
			// must be ordered oldest to newest!
			time.Date(2021, 1, 15, 12, 0, 0, 0, schema.BostonTime),
			time.Date(2022, 1, 14, 12, 0, 0, 0, schema.BostonTime),
			time.Date(2023, 1, 13, 12, 0, 0, 0, schema.BostonTime),
			time.Date(2023, 1, 13, 12, 0, 0, 0, schema.BostonTime),
			time.Date(2024, 1, 12, 12, 0, 0, 0, schema.BostonTime),
		},
	}
}

func (b *HuntYetBot) Register() (*discordgo.ApplicationCommand, bool) {
	return &discordgo.ApplicationCommand{
		Name:        "huntyet",
		Description: "IS IT HUNT YET??? ⏳",
	}, false
}

func (b *HuntYetBot) Handle(ctx context.Context, s *discordgo.Session,
	i *discord.CommandInput) (string, error) {

	var now = time.Now()
	for _, start := range b.hunts {
		end := start.Add(b.duration)
		if now.Before(start) {
			return fmt.Sprintf("No. You'll have to wait another %v.", b.formatDuration(start.Sub(now))), nil
		} else if now.Before(end) {
			return "Yes! HUNT HUNT HUNT!", nil
		}
		// else: this hunt has passed, check the next hunt
	}
	return "Gosh, I'm not sure! @tech can update the bot.", nil
}

func (b *HuntYetBot) formatDuration(d time.Duration) string {
	return fmt.Sprintf(
		"%d days, %v hours, %v minutes, %v.%02d seconds",
		int(d.Hours())/24,
		int(d.Hours())%24,
		int(d.Minutes())%60,
		int(d.Seconds())%60,
		int(d.Milliseconds())%1000,
	)
}
