package syncer

import (
	"context"
	"fmt"

	"github.com/gauravjsingh/emojihunt/schema"
)

func (s *Syncer) IdempotentUpdate(ctx context.Context, puzzle *schema.Puzzle) (*schema.Puzzle, error) {
	if puzzle.Status == puzzle.LastBotStatus && puzzle.ShouldArchive() == puzzle.Archived {
		// Nothing to do!
		return puzzle, nil
	}

	// Update channel and spreadsheet, if required
	if err := s.discordCreateUpdatePin(puzzle); err != nil {
		return nil, fmt.Errorf("unable to set puzzle status message for %q: %w", puzzle.Name, err)
	}

	if err := s.discordUpdateChannelCategory(puzzle); err != nil {
		return nil, fmt.Errorf("unable to set channel category for %q: %v", puzzle.Name, err)
	}

	if err := s.driveUpdateSpreadsheet(ctx, puzzle); err != nil {
		return nil, fmt.Errorf("unable to update spreadsheet title and folder for %q: %v", puzzle.Name, err)
	}

	// Send notifications
	var err error
	if puzzle.Status.IsSolved() {
		if puzzle.Answer != "" {
			// Puzzle solved and answer entered!
			err = s.notifyPuzzleFullySolved(puzzle)
		} else {
			// Puzzle marked as solved but answer needs to be entered in
			// Airtable...
			err = s.notifyPuzzleSolvedMissingAnswer(puzzle)
		}
	} else {
		// Ordinary status change
		err = s.notifyPuzzleStatusChange(puzzle)
	}
	if err != nil {
		return nil, fmt.Errorf("error posting puzzle status announcement: %v", err)
	}

	// Update bot status in Airtable
	puzzle, err = s.airtable.UpdateBotFields(puzzle, puzzle.Status, puzzle.ShouldArchive())
	if err != nil {
		return nil, fmt.Errorf("failed to update bot fields for puzzle %q: %v", puzzle.Name, err)
	}

	return puzzle, nil
}
