package syncer

import (
	"context"
	"log"
	"sync"

	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/db/field"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/drive"
	"golang.org/x/xerrors"
)

type Syncer struct {
	db      *db.Client
	discord *discord.Client
	drive   *drive.Client

	VoiceRoomMutex       sync.Mutex
	DiscordCategoryMutex sync.Mutex
}

func New(db *db.Client, discord *discord.Client, drive *drive.Client) *Syncer {
	return &Syncer{
		db:      db,
		discord: discord,
		drive:   drive,
	}
}

// IdempotentCreateUpdate synchronizes Discord and Google Drive with the puzzle
// information in the database. When a puzzle is newly added, it creates the
// spreadsheet and Discord channel and stores their IDs in the database. When a
// puzzle's status is updated, it handles that also.
func (s *Syncer) IdempotentCreateUpdate(ctx context.Context, puzzle *db.Puzzle) (*db.Puzzle, error) {
	// 1. Create the spreadsheet, if required
	if puzzle.SpreadsheetID == "" {
		spreadsheet, err := s.drive.CreateSheet(ctx, puzzle.Name, puzzle.Round.Name)
		if err != nil {
			return nil, err
		}

		puzzle, err = s.db.SetSpreadsheetID(ctx, puzzle, spreadsheet)
		if err != nil {
			return nil, err
		}

		err = s.driveUpdateSpreadsheet(ctx, puzzle)
		if err != nil {
			return nil, err
		}
	}

	// 2. Create the Discord channel, if required
	if puzzle.DiscordChannel == "" {
		log.Printf("Adding channel for new puzzle %q", puzzle.Name)
		category, err := s.discordGetOrCreateCategory(puzzle)
		if err != nil {
			return nil, err
		}

		channel, err := s.discord.CreateChannel(puzzle.Name, category)
		if err != nil {
			return nil, err
		}

		puzzle, err = s.db.SetDiscordChannel(ctx, puzzle, channel.ID)
		if err != nil {
			return nil, err
		}

		err = s.DiscordCreateUpdatePin(puzzle)
		if err != nil {
			return nil, err
		}

		if err := s.discordUpdateChannel(puzzle); err != nil {
			return nil, err
		}

		// Treat Discord channel creation as the sentinel to also notify the
		// team about the new puzzle.
		if err := s.notifyNewPuzzle(puzzle); err != nil {
			return nil, err
		}
	}

	// 3. Update the spreadsheet and Discord channel with new information, if required
	if puzzle.Status.IsSolved() != puzzle.Archived {
		var err error
		puzzle, err = s.HandleStatusChange(ctx, puzzle, false)
		if err != nil {
			return nil, err
		}
	}

	return puzzle, nil
}

// HandleStatusChange synchronizes Discord and sends notifications when the
// puzzle status changes. It's called by IdempotentCreateUpdate. If you're
// calling this from a slash command handler, and the is going to acknowledge
// the user in the puzzle channel, you can set `botRequest=true` to suppress
// notifications to the puzzle channel.
func (s *Syncer) HandleStatusChange(
	ctx context.Context, puzzle *db.Puzzle, botRequest bool,
) (*db.Puzzle, error) {
	log.Printf("syncer: handling status change for %q", puzzle.Name)
	if puzzle.SpreadsheetID == "-" || puzzle.DiscordChannel == "-" {
		return nil, xerrors.Errorf("puzzle is a placeholder puzzle, skipping")
	}

	var err error
	err = s.parallelHardUpdate(ctx, puzzle)
	if err != nil {
		return nil, err
	}

	// Update bot status in Airtable, unless we're in a bot handler and this has
	// already been done.
	if !botRequest {
		puzzle, err = s.db.SetBotFields(ctx, puzzle)
		if err != nil {
			return nil, err
		}
	}

	// Send notifications
	if puzzle.Status.IsSolved() {
		// Puzzle solved and answer entered! (Suppress puzzle channel
		// notification if this is a bot request, since the bot will also
		// respond in the puzzle channel.)
		err = s.notifyPuzzleFullySolved(puzzle, botRequest)
		if err != nil {
			return nil, err
		}

		// Also unset the voice room, if applicable
		if puzzle.VoiceRoom != "" {
			s.VoiceRoomMutex.Lock()
			defer s.VoiceRoomMutex.Unlock()
			puzzle, err = s.db.SetVoiceRoom(ctx, puzzle, nil)
			if err != nil {
				return nil, err
			}
			if err = s.DiscordCreateUpdatePin(puzzle); err != nil {
				return nil, err
			}
			if err = s.SyncVoiceRooms(ctx); err != nil {
				return nil, err
			}
		}
	} else if puzzle.Status == field.StatusWorking {
		if err = s.notifyPuzzleWorking(puzzle); err != nil {
			return nil, err
		}
	}
	return puzzle, nil
}

// ForceUpdate is a big hammer that will update Discord and Google Drive,
// including overwriting the channel name, spreadsheet name, etc. It also
// re-sends any status change notifications.
func (s *Syncer) ForceUpdate(ctx context.Context, puzzle *db.Puzzle) (*db.Puzzle, error) {
	if puzzle.SpreadsheetID == "-" || puzzle.DiscordChannel == "-" {
		return nil, xerrors.Errorf("puzzle is a placeholder puzzle, skipping")
	}

	var err error
	puzzle, err = s.IdempotentCreateUpdate(ctx, puzzle)
	if err != nil {
		return nil, err
	}

	err = s.parallelHardUpdate(ctx, puzzle)
	if err != nil {
		return nil, err
	}

	// Update bot status in Airtable
	puzzle, err = s.db.SetBotFields(ctx, puzzle)
	if err != nil {
		return nil, err
	}
	return puzzle, nil
}

// parallelHardUpdate updates the Discord pinned message, the Discord channel
// name/category, and the Google spreadsheet name. It's called when the puzzle
// status changes, and as part of ForceUpdate().
func (s *Syncer) parallelHardUpdate(ctx context.Context, puzzle *db.Puzzle) error {
	var wg sync.WaitGroup
	var ch = make(chan error, 3)
	wg.Add(3)
	go func() { ch <- s.DiscordCreateUpdatePin(puzzle); wg.Done() }()
	go func() { ch <- s.discordUpdateChannel(puzzle); wg.Done() }()
	go func() { ch <- s.driveUpdateSpreadsheet(ctx, puzzle); wg.Done() }()
	wg.Wait()
	close(ch)
	for err := range ch {
		if err != nil {
			return err
		}
	}
	return nil
}
