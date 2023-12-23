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
// Puzzle object.
func (c *Client) CreateDiscordChannel(ctx context.Context, puzzle state.Puzzle) (state.Puzzle, error) {
	log.Printf("sync: creating discord channel for %q", puzzle.Name)
	channel, err := c.discord.CreateChannel(puzzle.Name, puzzle.Round.DiscordCategory)
	if err != nil {
		return puzzle, err
	}
	return c.state.UpdatePuzzleAdvanced(ctx, puzzle.ID,
		func(puzzle *state.RawPuzzle) error {
			if puzzle.DiscordChannel != "" {
				return xerrors.Errorf("created duplicate Discord channel")
			}
			puzzle.DiscordChannel = channel.ID
			return nil
		}, false,
	)
}

// CreateDiscordCategory creates a new Discord category and saves it to the
// Round object.
func (c *Client) CreateDiscordCategory(ctx context.Context, round state.Round) (state.Round, error) {
	log.Printf("sync: creating discord category for %q", round.Name)
	category, err := c.discord.CreateCategory(roundCategoryPrefix + round.Name)
	if err != nil {
		return round, err
	}
	return c.state.UpdateRoundAdvanced(ctx, round.ID,
		func(round *state.Round) error {
			if round.DiscordCategory != "" {
				return xerrors.Errorf("created duplicate Discord category")
			}
			round.DiscordCategory = category.ID
			return nil
		}, false,
	)
}

func (c *Client) RestoreSolvedCategories() error {
	categories, err := c.discord.GetChannelCategories()
	if err != nil {
		return err
	}
	var solved []string
	for i := 0; i < solvedCategoryCount; i++ {
		name := solvedCategoryPrefix + string(rune(int('A')+i))
		if category, ok := categories[name]; ok {
			solved = append(solved, category.ID)
		} else {
			category, err := c.discord.CreateCategory(name)
			if err != nil {
				return err
			}
			solved = append(solved, category.ID)
		}
	}
	c.solvedCategories = solved
	return nil
}

type DiscordChannelFields struct {
	PuzzleName    string
	PuzzleChannel string
	RoundName     string
	RoundCategory string
	IsSolved      bool
}

func NewDiscordChannelFields(puzzle state.Puzzle) DiscordChannelFields {
	return DiscordChannelFields{
		PuzzleName:    puzzle.Name,
		PuzzleChannel: puzzle.DiscordChannel,
		RoundName:     puzzle.Round.Name,
		RoundCategory: puzzle.Round.DiscordCategory,
		IsSolved:      puzzle.Status.IsSolved(),
	}
}

// UpdateDiscordChannel configures the name and category of the puzzle channel.
// Categories are either in a round-specific category (if unsolved) or one of a
// few "Solved" categories (for solved puzzles), and the channel name is
// prefixed with a check mark when the puzzle is solved.
func (c *Client) UpdateDiscordChannel(fields DiscordChannelFields) error {
	log.Printf("sync: updating discord channel for %q", fields.PuzzleName)

	// Move puzzle channel to the correct category
	var category = fields.RoundCategory
	if fields.IsSolved {
		h := sha256.New()
		if _, err := h.Write([]byte(fields.PuzzleChannel)); err != nil {
			return err
		}
		i := binary.BigEndian.Uint64(h.Sum(nil)[:8]) % solvedCategoryCount
		category = c.solvedCategories[i]
	}
	err := c.discord.SetChannelCategory(fields.PuzzleChannel, category)
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
		ch <- c.discord.SetChannelName(fields.PuzzleChannel, title)
	}()
	select {
	case err := <-ch:
		return err
	case <-time.After(5 * time.Second):
		rateLimit := c.discord.CheckRateLimit(discordgo.EndpointChannel(fields.PuzzleChannel))
		if rateLimit == nil {
			// No rate limiting detected; maybe the Discord request is just
			// slow? Wait for it to finish.
			return <-ch
		}
		// Being rate limited; goroutine will finish later.
		msg := fmt.Sprintf(":snail: Hit Discord's rate limit on channel renaming. Channel will be "+
			"renamed to %q in %s.", title, time.Until(*rateLimit).Round(time.Second))
		return c.discord.ChannelSendRawID(fields.PuzzleChannel, msg)
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
func (c *Client) UpdateDiscordPin(fields DiscordPinFields) error {
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
		if fields.Location != "" {
			if locationMsg != locationDefaultMsg {
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

	return c.discord.CreateUpdatePin(fields.DiscordChannel, pinnedStatusHeader, embed)
}
