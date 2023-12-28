package sync

import (
	"context"
	"log"

	"github.com/emojihunt/emojihunt/state"
)

// CreateSpreadsheet creates a new Google Sheets spreadsheet and returns its ID.
func (c *Client) CreateSpreadsheet(ctx context.Context, puzzle state.RawPuzzle) (string, error) {
	log.Printf("sync: creating spreadsheet for %q", puzzle.Name)
	return c.drive.CreateSheet(ctx, puzzle.Name)
}

// CreateDriveFolder creates a new Google Drive folder and returns its ID.
func (c *Client) CreateDriveFolder(ctx context.Context, round state.Round) (string, error) {
	log.Printf("sync: creating drive folder for %q", round.Name)
	return c.drive.CreateFolder(ctx, round.Name)
}

type SpreadsheetFields struct {
	PuzzleName       string
	SpreadsheetID    string
	RoundName        string
	RoundDriveFolder string
	IsSolved         bool
}

func NewSpreadsheetFields(puzzle state.Puzzle) SpreadsheetFields {
	return SpreadsheetFields{
		SpreadsheetID:    puzzle.SpreadsheetID,
		PuzzleName:       puzzle.Name,
		RoundName:        puzzle.Round.Name,
		RoundDriveFolder: puzzle.Round.DriveFolder,
		IsSolved:         puzzle.Status.IsSolved(),
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
	return c.drive.SetSheetFolder(ctx, fields.SpreadsheetID, fields.RoundDriveFolder)
}
