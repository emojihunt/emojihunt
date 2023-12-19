package bot

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/huntyet"
)

type HuntYetBot struct{}

func NewHuntYetBot() discord.Bot {
	return &HuntYetBot{}
}

func (b *HuntYetBot) Register() (*discordgo.ApplicationCommand, bool) {
	return &discordgo.ApplicationCommand{
		Name:        "huntyet",
		Description: "IS IT HUNT YET??? ⏳",
	}, false
}

func (b *HuntYetBot) Handle(ctx context.Context, input *discord.CommandInput) (string, error) {
	var now = time.Now()
	next, ok := huntyet.NextHunt(now)
	if !ok {
		return "Gosh, I'm not sure! @tech can update the bot.", nil
	} else if next == nil {
		return "Yes! HUNT HUNT HUNT!", nil
	} else {
		return fmt.Sprintf("No. You'll have to wait another %v.", b.formatDuration(next.Sub(now))), nil
	}
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

func (b *HuntYetBot) HandleScheduledEvent(context.Context,
	*discordgo.GuildScheduledEventUpdate) error {
	return nil
}
