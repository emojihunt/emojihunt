package syncer

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/schema"
)

const (
	roundCategoryPrefix   = "Round: "
	solvedCategoryPrefix  = "Solved "
	pinnedStatusHeader    = "Puzzle Information"
	locationDefaultMsg    = "Use `/voice start` to assign a voice room"
	embedColor            = 0x7C39ED
	backgroundTaskTimeout = 120 * time.Second
)

// DiscordCreateUpdatePin creates or updates the pinned message at the top of
// the puzzle channel. This message contains information about the puzzle status
// as well as links to the puzzle and the spreadsheet.
//
// This function is called by BasicUpdate. Other packages need to call it when
// updating non-status fields, such as the voice room.
func (s *Syncer) DiscordCreateUpdatePin(puzzle *schema.Puzzle) error {
	log.Printf("syncer: updating pin for %q", puzzle.Name)

	roundHeader := "Round"
	if len(puzzle.Rounds) > 1 {
		roundHeader = "Rounds"
	}
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: pinnedStatusHeader},
		Title:  puzzle.Name,
		URL:    puzzle.PuzzleURL,
		Color:  embedColor,
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

	if !puzzle.Status.IsSolved() {
		locationMsg := locationDefaultMsg
		if puzzle.VoiceRoom != "" {
			locationMsg = fmt.Sprintf("Join us in <#%s>!", puzzle.VoiceRoom)
		} else if puzzle.Location != "" {
			locationMsg = fmt.Sprintf("In-person in %s", puzzle.Location)
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Location",
			Value:  locationMsg,
			Inline: false,
		})
	}

	return s.discord.CreateUpdatePin(puzzle.DiscordChannel, pinnedStatusHeader, embed)
}

// discordUpdateChannel sets or updates the name and category of the puzzle
// channel. Categories are either "Puzzles" (for open puzzles) or "Solved" (for
// solved puzzles), and the puzzle name includes a check mark when the puzzle is
// solved. It needs to be called when the puzzle status changes.
func (s *Syncer) discordUpdateChannel(puzzle *schema.Puzzle) error {
	log.Printf("syncer: updating discord channel for %q", puzzle.Name)

	// Move puzzle channel to the correct category
	category, err := s.discordGetOrCreateCategory(puzzle)
	if err != nil {
		return err
	}
	err = s.discord.SetChannelCategory(puzzle.DiscordChannel, category)
	if err != nil {
		return err
	}

	// The Discord rate limit on channel renames is fairly restrictive (2 per 10
	// minutes per channel), so finish renaming the channel asynchronously if we
	// get rate-limited.
	var title = puzzle.Title()
	if puzzle.Status.IsSolved() {
		title = "✅ " + title
	}
	ch := make(chan error)
	go func() {
		_, cancel := context.WithTimeout(s.main, backgroundTaskTimeout)
		defer cancel()

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
		// Being rate limited; goroutine will finish later.
		msg := fmt.Sprintf(":snail: Hit Discord's rate limit on channel renaming. Channel will be "+
			"renamed to %q in %s.", title, time.Until(*rateLimit).Round(time.Second))
		return s.discord.ChannelSendRawID(puzzle.DiscordChannel, msg)
	}
}

func (s *Syncer) discordGetOrCreateCategory(puzzle *schema.Puzzle) (*discordgo.Channel, error) {
	s.DiscordCategoryMutex.Lock()
	defer s.DiscordCategoryMutex.Unlock()

	categories, err := s.discord.GetChannelCategories()
	if err != nil {
		return nil, err
	}

	var targetName string
	if puzzle.ShouldArchive() {
		targetName = solvedCategoryPrefix + puzzle.ArchiveCategory()
	} else {
		targetName = roundCategoryPrefix + puzzle.Rounds[0].Name
	}

	if item, ok := categories[targetName]; !ok {
		// We need to create the category
		log.Printf("syncer: creating discord category %q", targetName)
		return s.discord.CreateCategory(targetName)
	} else {
		// Cateory already exists
		return item, nil
	}
}
