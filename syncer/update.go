package syncer

import (
	"context"
	"fmt"

	"github.com/gauravjsingh/emojihunt/schema"
)

func (s *Syncer) ProcessStatusUpdate(ctx context.Context, puzzle *schema.Puzzle) error {
	if puzzle.Status.IsSolved() {
		if err := s.MarkSolved(ctx, puzzle); err != nil {
			return fmt.Errorf("failed to mark puzzle %q solved: %v", puzzle.Name, err)
		}
	} else {
		didUpdate, err := s.SetPinnedStatusInfo(puzzle)
		if err != nil {
			return fmt.Errorf("unable to set puzzle status message for %q: %w", puzzle.Name, err)
		}

		if didUpdate {
			if err := s.discord.StatusUpdateChannelSend(fmt.Sprintf("%s Puzzle <#%s> is now %v.", puzzle.Round.Emoji, puzzle.DiscordChannel, puzzle.Status.Pretty())); err != nil {
				return fmt.Errorf("error posting puzzle status announcement: %v", err)
			}
		}
	}
	return nil
}
