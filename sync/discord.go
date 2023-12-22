package sync

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/state/status"
	"golang.org/x/xerrors"
)

const (
	roundCategoryPrefix  = "Round: "
	solvedCategoryPrefix = "Solved "
	solvedCategoryCount  = 3
	pinnedStatusHeader   = "Puzzle Information"
	locationDefaultMsg   = "Use `/puzzle voice` to assign a voice room"
	embedColor           = 0x7C39ED
)

// CreateDiscordChannel creates a new Discord channel and saves it to the
// database.
func (c *Client) CreateDiscordChannel(ctx context.Context, puzzle state.Puzzle) (state.Puzzle, error) {
	log.Printf("sync: creating discord channel for %q", puzzle.Name)
	category, err := c.GetCreateDiscordCategory(
		puzzle.DiscordChannel, puzzle.Round.Name, false)
	if err != nil {
		return state.Puzzle{}, err
	}

	channel, err := c.discord.CreateChannel(puzzle.Name, category)
	if err != nil {
		return state.Puzzle{}, err
	}

	// TODO: don't trigger an infinite loop
	puzzle, err = c.state.UpdatePuzzle(ctx, puzzle.ID,
		func(puzzle *state.RawPuzzle) error {
			if puzzle.DiscordChannel != "" {
				return xerrors.Errorf("created duplicate Discord channel")
			}
			puzzle.DiscordChannel = channel.ID
			return nil
		},
	)
	if err != nil {
		return state.Puzzle{}, err
	}
	return puzzle, nil
}

type DiscordChannelFields struct {
	DiscordChannel string
	PuzzleName     string
	RoundName      string
	IsSolved       bool
}

func NewDiscordChannelFields(puzzle state.Puzzle) DiscordChannelFields {
	return DiscordChannelFields{
		DiscordChannel: puzzle.DiscordChannel,
		PuzzleName:     puzzle.Name,
		RoundName:      puzzle.Round.Name,
		IsSolved:       puzzle.Status.IsSolved(),
	}
}

// UpdateDiscordChannel configures the name and category of the puzzle channel.
// Categories are either in a round-specific category (if unsolved) or one of a
// few "Solved" categories (for solved puzzles), and the channel name is
// prefixed with a check mark when the puzzle is solved.
func (c *Client) UpdateDiscordChannel(fields DiscordChannelFields) error {
	log.Printf("sync: updating discord channel for %q", fields.PuzzleName)

	// Move puzzle channel to the correct category
	category, err := c.GetCreateDiscordCategory(
		fields.DiscordChannel, fields.RoundName, fields.IsSolved)
	if err != nil {
		return err
	}
	err = c.discord.SetChannelCategory(fields.DiscordChannel, category)
	if err != nil {
		return err
	}

	// The Discord rate limit on channel renames is fairly restrictive (2 per 10
	// minutes per channel), so finish renaming the channel asynchronously if we
	// get rate-limited.
	var title = fields.PuzzleName
	if fields.IsSolved {
		title = "✅ " + title
	}
	ch := make(chan error)
	go func() {
		ch <- c.discord.SetChannelName(fields.DiscordChannel, title)
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(5 * time.Second):
		rateLimit := c.discord.CheckRateLimit(discordgo.EndpointChannel(fields.DiscordChannel))
		if rateLimit == nil {
			// No rate limiting detected; maybe the Discord request is just
			// slow? Wait for it to finish.
			return <-ch
		}
		// Being rate limited; goroutine will finish later.
		msg := fmt.Sprintf(":snail: Hit Discord's rate limit on channel renaming. Channel will be "+
			"renamed to %q in %s.", title, time.Until(*rateLimit).Round(time.Second))
		return c.discord.ChannelSendRawID(fields.DiscordChannel, msg)
	}
}

type DiscordPinFields struct {
	RoundName  string
	RoundEmoji string

	PuzzleName     string
	Status         status.Status
	Note           string
	Location       string
	PuzzleURL      string
	SpreadsheetID  string
	DiscordChannel string
	VoiceRoom      string
}

func NewDiscordPinFields(puzzle state.Puzzle) DiscordPinFields {
	var spreadsheet, channel string
	if puzzle.HasSpreadsheetID() {
		spreadsheet = puzzle.SpreadsheetID
	}
	if puzzle.HasDiscordChannel() {
		channel = puzzle.DiscordChannel
	}
	return DiscordPinFields{
		RoundName:      puzzle.Round.Name,
		RoundEmoji:     puzzle.Round.Emoji,
		PuzzleName:     puzzle.Name,
		Status:         puzzle.Status,
		Note:           puzzle.Note,
		Location:       puzzle.Location,
		PuzzleURL:      puzzle.PuzzleURL,
		SpreadsheetID:  spreadsheet,
		DiscordChannel: channel,
		VoiceRoom:      puzzle.VoiceRoom,
	}
}

// UpdateDiscordPin creates or updates the pinned message at the top of the
// puzzle channel. This message contains information about the puzzle status as
// well as links to the puzzle and the spreadsheet.
func (s *Client) UpdateDiscordPin(fields DiscordPinFields) error {
	log.Printf("sync: updating discord pin for %q", fields.PuzzleName)

	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: pinnedStatusHeader},
		Title:  fields.PuzzleName,
		URL:    fields.PuzzleURL,
		Color:  embedColor,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Round",
				Value:  fmt.Sprintf("%s %s", fields.RoundEmoji, fields.RoundName),
				Inline: false,
			},
			{
				Name:   "Status",
				Value:  fields.Status.Pretty(),
				Inline: true,
			},
			{
				Name:   "Puzzle",
				Value:  fmt.Sprintf("[Link](%s)", fields.PuzzleURL),
				Inline: true,
			},
		},
	}

	if fields.SpreadsheetID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "Sheet",
			Value: fmt.Sprintf("[Link](https://docs.google.com/spreadsheets/d/%s)",
				fields.SpreadsheetID),
			Inline: true,
		})
	}

	if fields.Note != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Note",
			Value:  fields.Note,
			Inline: false,
		})
	}

	if !fields.Status.IsSolved() {
		locationMsg := locationDefaultMsg
		if fields.VoiceRoom != "" {
			locationMsg = fmt.Sprintf("Join us in <#%s>!", fields.VoiceRoom)
		}
		if fields.Location != locationDefaultMsg {
			if locationMsg != "" {
				locationMsg += fmt.Sprintf("Also in-person in %s.", fields.Location)
			} else {
				locationMsg = fmt.Sprintf("In-person in %s.", fields.Location)
			}
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Location",
			Value:  locationMsg,
			Inline: false,
		})
	}

	return s.discord.CreateUpdatePin(fields.DiscordChannel, pinnedStatusHeader, embed)
}

// GetCreateDiscordCategory returns the appropriate Discord category for the
// puzzle given its current state, creating it if it doesn't already exist.
func (s *Client) GetCreateDiscordCategory(channel string, round string, solved bool) (*discordgo.Channel, error) {
	categories, err := s.discord.GetChannelCategories()
	if err != nil {
		return nil, err
	}

	var targetName string
	if solved {
		// Hash the Discord channel ID, since it's not totally random
		h := sha256.New()
		if _, err := h.Write([]byte(channel)); err != nil {
			return nil, err
		}
		i := binary.BigEndian.Uint64(h.Sum(nil)[:8]) % solvedCategoryCount
		targetName = solvedCategoryPrefix + string(rune(uint64('A')+i))
	} else {
		targetName = roundCategoryPrefix + round
	}

	if item, ok := categories[targetName]; !ok {
		// we need to create the category
		log.Printf("sync: creating discord category %q", targetName)
		return s.discord.CreateCategory(targetName)
	} else {
		// cateory already exists
		return item, nil
	}
}
