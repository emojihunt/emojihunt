package syncer

import (
	"context"
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
)

// IdempotentCreate creates the Google Sheet and Discord channel if needed, then
// notifies the team about it in #general. If the puzzle has already been set
// up, it's a complete no-op.
func (s *Syncer) IdempotentCreate(ctx context.Context, puzzle *schema.Puzzle) (*schema.Puzzle, error) {
	if puzzle.SpreadsheetID == "" {
		spreadsheet, err := s.drive.CreateSheet(ctx, puzzle.Name, puzzle.Round.Name)
		if err != nil {
			return nil, fmt.Errorf("error creating spreadsheet for %q: %v", puzzle.Name, err)
		}
		puzzle, err = s.airtable.UpdateSpreadsheetID(puzzle, spreadsheet)
		if err != nil {
			return nil, fmt.Errorf("error setting spreadsheet id for puzzle %q: %v", puzzle.Name, err)
		}
	}

	if puzzle.DiscordChannel == "" {
		log.Printf("Adding channel for new puzzle %q", puzzle.Name)
		channel, err := s.discord.CreateChannel(puzzle.Name)
		if err != nil {
			return nil, fmt.Errorf("error creating discord channel for %q: %v", puzzle.Name, err)
		}

		puzzle, err = s.airtable.UpdateDiscordChannel(puzzle, channel)
		if err != nil {
			return nil, fmt.Errorf("error setting discord channel for puzzle %q: %v", puzzle.Name, err)
		}

		// Treat Discord channel creation as the sentinel to also notify the
		// team about the new puzzle.
		if err := s.notifyNewPuzzle(puzzle); err != nil {
			return nil, fmt.Errorf("error notifying channel about new puzzle %q: %v", puzzle.Name, err)
		}
	}

	return puzzle, nil
}

func (s *Syncer) notifyNewPuzzle(puzzle *schema.Puzzle) error {
	log.Printf("Posting information about new puzzle %q", puzzle.Name)

	// Pin a message with the spreadsheet URL to the channel
	if _, err := s.SetPinnedStatusInfo(puzzle); err != nil {
		return fmt.Errorf("error pinning puzzle info: %v", err)
	}

	// Post a message in the general channel with a link to the puzzle.
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "A new puzzle is available!",
			IconURL: puzzle.Round.TwemojiURL(),
		},
		Title: puzzle.Name,
		URL:   puzzle.PuzzleURL,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Channel",
				Value:  fmt.Sprintf("<#%s>", puzzle.DiscordChannel),
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
	if err := s.discord.GeneralChannelSendEmbed(embed); err != nil {
		return fmt.Errorf("error posting new puzzle announcement: %v", err)
	}

	return nil
}
