package syncer

import (
	"context"
	"log"

	"github.com/emojihunt/emojihunt/state"
)

// CreateSpreadsheet creates a new Google Sheets spreadsheet and returns its ID.
func (c *Client) CreateSpreadsheet(ctx context.Context, puzzle state.RawPuzzle, round state.Round) (string, error) {
	log.Printf("sync: creating spreadsheet for %q", puzzle.Name)
	return c.drive.CreateSheet(ctx, puzzle.Name, round.DriveFolder)
}

// CreateDriveFolder creates a new Google Drive folder and returns its ID.
func (c *Client) CreateDriveFolder(ctx context.Context, round state.Round) (string, error) {
	log.Printf("sync: creating drive folder for %q", round.Name)
	return c.drive.CreateFolder(ctx, round.Name)
}

type SpreadsheetFields struct {
	PuzzleName       string
	SpreadsheetID    string
	RoundDriveFolder string
}

func NewSpreadsheetFields(puzzle state.Puzzle) SpreadsheetFields {
	return SpreadsheetFields{
		SpreadsheetID:    puzzle.SpreadsheetID,
		PuzzleName:       puzzle.Name,
		RoundDriveFolder: puzzle.Round.DriveFolder,
	}
}

// UpdateSpreadsheet sets the spreadsheet's title and parent folder. The title
// is the name of the puzzle, plus a check mark if the puzzle has been solved.
// The folder is based on the round.
func (c *Client) UpdateSpreadsheet(ctx context.Context, fields SpreadsheetFields) error {
	log.Printf("sync: updating spreadsheet for %q", fields.PuzzleName)
	err := c.drive.SetSheetTitle(ctx, fields.SpreadsheetID, fields.PuzzleName)
	if err != nil {
		return err
	}
	return c.drive.SetSheetFolder(ctx, fields.SpreadsheetID, fields.RoundDriveFolder)
}

type DriveFolderFields struct {
	RoundName        string
	RoundDriveFolder string
}

func NewDriveFolderFields(round state.Round) DriveFolderFields {
	return DriveFolderFields{
		RoundName:        round.Name,
		RoundDriveFolder: round.DriveFolder,
	}
}

// UpdateDriveFolder sets the Google Drive folder name.
func (c *Client) UpdateDriveFolder(ctx context.Context, fields DriveFolderFields) error {
	log.Printf("sync: updating drive folder for %q", fields.RoundName)
	return c.drive.SetFolderName(ctx, fields.RoundDriveFolder, fields.RoundName)
}
