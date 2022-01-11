package syncer

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
)

const (
	pinnedStatusHeader  = "Puzzle Information"
	voiceRoomDefaultMsg = "Use `/voice start` to assign a voice room"
)

// DiscordCreateUpdatePin creates or updates the pinned message at the top of
// the puzzle channel. This message contains information about the puzzle status
// as well as links to the puzzle and the spreadsheet.
//
// This function is called by BasicUpdate. Other packages need to call it when
// updating non-status fields, such as the voice room.
//
func (s *Syncer) DiscordCreateUpdatePin(puzzle *schema.Puzzle) error {
	roundHeader := "Round"
	if len(puzzle.Rounds) > 1 {
		roundHeader = "Rounds"
	}
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: pinnedStatusHeader},
		Title:  puzzle.Name,
		URL:    puzzle.PuzzleURL,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   roundHeader,
				Value:  strings.Join(puzzle.Rounds.EmojisAndNames(), ", "),
				Inline: false,
			},
			{
				Name:   "Status",
				Value:  puzzle.Status.Human(),
				Inline: true,
			},
			{
				Name:   "Puzzle",
				Value:  fmt.Sprintf("[Link](%s)", puzzle.PuzzleURL),
				Inline: true,
			},
			{
				Name:   "Sheet",
				Value:  fmt.Sprintf("[Link](%s)", puzzle.SpreadsheetURL()),
				Inline: true,
			},
		},
	}

	if puzzle.Description != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Description",
			Value:  puzzle.Description,
			Inline: false,
		})
	}

	if puzzle.Notes != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Notes",
			Value:  puzzle.Notes,
			Inline: false,
		})
	}

	voiceRoomMsg := voiceRoomDefaultMsg
	if puzzle.VoiceRoomEvent != "" {
		var err error
		event, err := s.discord.GetScheduledEvent(puzzle.VoiceRoomEvent)
		if err != nil {
			return err
		}
		voiceRoomMsg = fmt.Sprintf("Join us in <#%s>!", *event.ChannelID)
	}
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "Voice Room",
		Value:  voiceRoomMsg,
		Inline: false,
	})

	return s.discord.CreateUpdatePin(puzzle.DiscordChannel, pinnedStatusHeader, embed)
}

// discordUpdateChannel sets or updates the name and category of the puzzle
// channel. Categories are either "Puzzles" (for open puzzles) or "Solved" (for
// solved puzzles), and the puzzle name includes a check mark when the puzzle is
// solved. It needs to be called when the puzzle status changes.
func (s *Syncer) discordUpdateChannel(puzzle *schema.Puzzle) error {
	var category *discordgo.Channel
	if puzzle.ShouldArchive() {
		category = s.discord.SolvedCategory
	} else {
		category = s.discord.PuzzleCategory
	}

	err := s.discord.SetChannelCategory(puzzle.DiscordChannel, category)
	if err != nil {
		return err
	}

	var title = puzzle.Title()
	if puzzle.Status.IsSolved() {
		title = "✅ " + title
	}

	// The Discord rate limit on channel renames is fairly restrictive (2 per 10
	// minutes per channel), so finish renaming the channel asynchronously if we
	// get rate-limited.
	ch := make(chan error)
	go func() {
		ch <- s.discord.SetChannelName(puzzle.DiscordChannel, title)
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(5 * time.Second):
		rateLimit := s.discord.CheckRateLimit(discordgo.EndpointChannel(puzzle.DiscordChannel))
		if rateLimit == nil {
			// No rate limiting detected; maybe the Discord request is just
			// slow? Wait for it to finish.
			return <-ch
		}
		// Being rate limited; goroutine will finish later. (TODO: this can
		// still result in an incorrect result if multiple channel renames are
		// blocked on the same rate limit -- which order they resolve in is
		// arbitrary.)
		msg := fmt.Sprintf(":snail: Hit Discord's rate limit on channel renaming. Channel will be "+
			"renamed to %q in %s.", title, rateLimit.Sub(time.Now()).Round(time.Second))
		return s.discord.ChannelSendRawID(puzzle.DiscordChannel, msg)
	}
}
