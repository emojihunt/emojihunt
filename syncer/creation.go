package syncer

import (
	"context"
	"fmt"
	"log"

	"github.com/gauravjsingh/emojihunt/schema"
)

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
		if err := s.NotifyNewPuzzle(puzzle); err != nil {
			return nil, fmt.Errorf("error notifying channel about new puzzle %q: %v", puzzle.Name, err)
		}
	}

	return puzzle, nil
}
