package syncer

import (
	"context"

	"github.com/gauravjsingh/emojihunt/schema"
)

// driveUpdateSpreadsheet sets the spreadsheet's title and parent folder. The
// title is the name of the puzzle, plus a check mark if the puzzle has been
// solved (so this function needs to be called when the puzzle's status is
// updated). The folder is based on the round, which shouldn't change after
// creation but we update it to be sure.
func (s *Syncer) driveUpdateSpreadsheet(ctx context.Context, puzzle *schema.Puzzle) error {
	var name = puzzle.Name
	if puzzle.Status.IsSolved() {
		name = "[SOLVED] " + name
	}
	if err := s.drive.SetSheetTitle(ctx, puzzle.SpreadsheetID, name); err != nil {
		return err
	}

	return s.drive.SetSheetFolder(ctx, puzzle.SpreadsheetID, puzzle.Rounds[0].Name)
}
