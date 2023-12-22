package sync

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/state"
)

const (
	roundCategoryPrefix  = "Round: "
	solvedCategoryPrefix = "Solved "
	solvedCategoryCount  = 3
	pinnedStatusHeader   = "Puzzle Information"
	locationDefaultMsg   = "Use `/puzzle voice` to assign a voice room"
	embedColor           = 0x7C39ED
)

// UpdateDiscordPin creates or updates the pinned message at the top of the
// puzzle channel. This message contains information about the puzzle status as
// well as links to the puzzle and the spreadsheet.
func (s *Client) UpdateDiscordPin(puzzle state.Puzzle) error {
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
		}
		if puzzle.Location != locationDefaultMsg {
			if locationMsg != "" {
				locationMsg += fmt.Sprintf("Also in-person in %s.", puzzle.Location)
			} else {
				locationMsg = fmt.Sprintf("In-person in %s.", puzzle.Location)
			}
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Location",
			Value:  locationMsg,
			Inline: false,
		})
	}

	return s.discord.CreateUpdatePin(puzzle.DiscordChannel, pinnedStatusHeader, embed)
}

// UpdateDiscordChannel configures the name and category of the puzzle channel.
// Categories are either in a round-specific category (if unsolved) or one of a
// few "Solved" categories (for solved puzzles), and the channel name is
// prefixed with a check mark when the puzzle is solved.
func (s *Client) UpdateDiscordChannel(puzzle state.Puzzle) error {
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

func (s *Client) discordGetOrCreateCategory(puzzle state.Puzzle) (*discordgo.Channel, error) {
	s.DiscordCategoryMutex.Lock()
	defer s.DiscordCategoryMutex.Unlock()

	categories, err := s.discord.GetChannelCategories()
	if err != nil {
		return nil, err
	}

	var targetName string
	if puzzle.Status.IsSolved() {
		// Hash the Discord channel ID, since it's not totally random
		h := sha256.New()
		if _, err := h.Write([]byte(puzzle.DiscordChannel)); err != nil {
			return nil, err
		}
		i := binary.BigEndian.Uint64(h.Sum(nil)[:8]) % solvedCategoryCount
		targetName = solvedCategoryPrefix + string(rune(uint64('A')+i))
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
