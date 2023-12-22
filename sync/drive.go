package sync

import (
	"context"
	"log"

	"github.com/emojihunt/emojihunt/state"
	"golang.org/x/xerrors"
)

// CreateSpreadsheet creates a new spreadsheet and saves it to the database.
func (c *Client) CreateSpreadsheet(ctx context.Context, puzzle state.Puzzle) (state.Puzzle, error) {
	log.Printf("sync: creating spreadsheet for %q", puzzle.Name)
	spreadsheet, err := c.drive.CreateSheet(ctx, puzzle.Name, puzzle.Round.Name)
	if err != nil {
		return state.Puzzle{}, err
	}

	// TODO: don't trigger an infinite loop
	puzzle, err = c.state.UpdatePuzzle(ctx, puzzle.ID,
		func(puzzle *state.RawPuzzle) error {
			if puzzle.SpreadsheetID != "" {
				return xerrors.Errorf("created duplicate spreadsheet")
			}
			puzzle.SpreadsheetID = spreadsheet
			return nil
		},
	)
	if err != nil {
		return state.Puzzle{}, err
	}

	err = c.UpdateSpreadsheet(ctx, puzzle)
	if err != nil {
		return state.Puzzle{}, err
	}
	return puzzle, nil
}

// UpdateSpreadsheet sets the spreadsheet's title and parent folder. The title
// is the name of the puzzle, plus a check mark if the puzzle has been solved.
// The folder is based on the round.
func (c *Client) UpdateSpreadsheet(ctx context.Context, puzzle state.Puzzle) error {
	log.Printf("sync: updating spreadsheet for %q", puzzle.Name)
	var name = puzzle.Name
	if puzzle.Status.IsSolved() {
		name = "✅ " + name
	}
	err := c.drive.SetSheetTitle(ctx, puzzle.SpreadsheetID, name)
	if err != nil {
		return err
	}
	return c.drive.SetSheetFolder(ctx, puzzle.SpreadsheetID, puzzle.Round.Name)
}
