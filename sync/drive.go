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
	return puzzle, nil
}

type SpreadsheetFields struct {
	SpreadsheetID string
	PuzzleName    string
	RoundName     string
	IsSolved      bool
}

func NewSpreadsheetFields(puzzle state.Puzzle) SpreadsheetFields {
	return SpreadsheetFields{
		SpreadsheetID: puzzle.SpreadsheetID,
		PuzzleName:    puzzle.Name,
		RoundName:     puzzle.Round.Name,
		IsSolved:      puzzle.Status.IsSolved(),
	}
}

// UpdateSpreadsheet sets the spreadsheet's title and parent folder. The title
// is the name of the puzzle, plus a check mark if the puzzle has been solved.
// The folder is based on the round.
func (c *Client) UpdateSpreadsheet(ctx context.Context, fields SpreadsheetFields) error {
	log.Printf("sync: updating spreadsheet for %q", fields.PuzzleName)
	var name = fields.PuzzleName
	if fields.IsSolved {
		name = "✅ " + name
	}
	err := c.drive.SetSheetTitle(ctx, fields.SpreadsheetID, name)
	if err != nil {
		return err
	}
	return c.drive.SetSheetFolder(ctx, fields.SpreadsheetID, fields.RoundName)
}
