package bot

import (
	"fmt"
	"strings"
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

func MakeHuntYetHandler() client.DiscordMessageHandler {
	return func(s *discordgo.Session, m *discordgo.MessageCreate) error {
		if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!huntyet") {
			return nil
		}

		var msg = "Gosh, I'm not sure! @tech can update the bot."
		var now = time.Now()
		for _, h := range hunts {
			if h.start.After(now) {
				msg = fmt.Sprintf("No. You'll have to wait another %v.", formatDuration(h.start.Sub(now)))
				break
			} else if h.end.After(now) {
				msg = "Yes! HUNT HUNT HUNT!"
				break
			}
		}
		_, err := s.ChannelMessageSend(m.ChannelID, msg)
		return err
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
