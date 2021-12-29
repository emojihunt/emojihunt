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
	var title = puzzle.Name
	if puzzle.Status.IsSolved() {
		title = "[SOLVED] " + title
	}
	if err := s.drive.SetSheetTitle(ctx, puzzle.SpreadsheetID, title); err != nil {
		return err
	}

	return s.drive.SetSheetFolder(ctx, puzzle.SpreadsheetID, puzzle.Round.Name)
}
