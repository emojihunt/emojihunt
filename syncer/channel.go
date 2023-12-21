package syncer

import (
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/state"
)

const (
	roundCategoryPrefix  = "Round: "
	solvedCategoryPrefix = "Solved "
	pinnedStatusHeader   = "Puzzle Information"
	locationDefaultMsg   = "Use `/voice start` to assign a voice room"
	embedColor           = 0x7C39ED
)

// DiscordCreateUpdatePin creates or updates the pinned message at the top of
// the puzzle channel. This message contains information about the puzzle status
// as well as links to the puzzle and the spreadsheet.
//
// This function is called by BasicUpdate. Other packages need to call it when
// updating non-status fields, such as the voice room.
func (s *Syncer) DiscordCreateUpdatePin(puzzle *state.Puzzle) error {
	log.Printf("syncer: updating pin for %q", puzzle.Name)

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: pinnedStatusHeader},
		Title:  puzzle.Name,
		URL:    puzzle.PuzzleURL,
		Color:  embedColor,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Round",
				Value:  fmt.Sprintf("%s %s", puzzle.Round.Emoji, puzzle.Round.Name),
				Inline: false,
			},
			{
				Name:   "Status",
				Value:  puzzle.Status.Pretty(),
				Inline: true,
			},
			{
				Name:   "Puzzle",
				Value:  fmt.Sprintf("[Link](%s)", puzzle.PuzzleURL),
				Inline: true,
			},
		},
	}

	if puzzle.SpreadsheetID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "Sheet",
			Value: fmt.Sprintf("[Link](https://docs.google.com/spreadsheets/d/%s)",
				puzzle.SpreadsheetID),
			Inline: true,
		})
	}

	if puzzle.Note != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Note",
			Value:  puzzle.Note,
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
func (s *Syncer) discordUpdateChannel(puzzle *state.Puzzle) error {
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
	var title = puzzle.Name
	if puzzle.Status.IsSolved() {
		title = "✅ " + title
	}
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
		// Being rate limited; goroutine will finish later.
		msg := fmt.Sprintf(":snail: Hit Discord's rate limit on channel renaming. Channel will be "+
			"renamed to %q in %s.", title, time.Until(*rateLimit).Round(time.Second))
		return s.discord.ChannelSendRawID(puzzle.DiscordChannel, msg)
	}
}

func (s *Syncer) discordGetOrCreateCategory(puzzle *state.Puzzle) (*discordgo.Channel, error) {
	s.DiscordCategoryMutex.Lock()
	defer s.DiscordCategoryMutex.Unlock()

	categories, err := s.discord.GetChannelCategories()
	if err != nil {
		return nil, err
	}

	var targetName string
	if puzzle.Status.IsSolved() {
		targetName = solvedCategoryPrefix + puzzle.ArchiveCategory()
	} else {
		targetName = roundCategoryPrefix + puzzle.Round.Name
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
