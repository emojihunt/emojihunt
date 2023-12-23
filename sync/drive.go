package sync

import (
	"context"
	"log"

	"github.com/emojihunt/emojihunt/state"
	"golang.org/x/xerrors"
)

// CreateSpreadsheet creates a new Google Sheets spreadsheet and saves it to the
// Puzzle object.
func (c *Client) CreateSpreadsheet(ctx context.Context, puzzle state.Puzzle) (state.Puzzle, error) {
	log.Printf("sync: creating spreadsheet for %q", puzzle.Name)
	spreadsheet, err := c.drive.CreateSheet(ctx, puzzle.Name)
	if err != nil {
		return state.Puzzle{}, err
	}

	return c.state.UpdatePuzzleAdvanced(ctx, puzzle.ID,
		func(puzzle *state.RawPuzzle) error {
			if puzzle.SpreadsheetID != "" {
				return xerrors.Errorf("created duplicate spreadsheet")
			}
			puzzle.SpreadsheetID = spreadsheet
			return nil
		}, false,
	)
}

// CreateDriveFolder creates a new Google Drive folder and saves it to the Round
// object.
func (c *Client) CreateDriveFolder(ctx context.Context, round state.Round) (state.Round, error) {
	log.Printf("sync: creating drive folder for %q", round.Name)
	folder, err := c.drive.CreateFolder(ctx, round.Name)
	if err != nil {
		return state.Round{}, err
	}
	return c.state.UpdateRoundAdvanced(ctx, round.ID,
		func(round *state.Round) error {
			if round.DriveFolder != "" {
				return xerrors.Errorf("created duplicate Google Drive folder")
			}
			round.DriveFolder = folder
			return nil
		}, false,
	)
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
