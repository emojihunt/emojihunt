package sync

import (
	"context"
	"log"
	"sync"

	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/drive"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/state/status"
	"golang.org/x/xerrors"
)

type Client struct {
	state   *state.Client
	discord *discord.Client
	drive   *drive.Client

	Discovery            bool
	VoiceRoomMutex       sync.Mutex
	DiscordCategoryMutex sync.Mutex
}

func New(discord *discord.Client, drive *drive.Client, state *state.Client, discovery bool) *Client {
	return &Client{
		Discovery: discovery,
		discord:   discord,
		drive:     drive,
		state:     state,
	}
}

// IdempotentCreateUpdate synchronizes Discord and Google Drive with the puzzle
// information in the database. When a puzzle is newly added, it creates the
// spreadsheet and Discord channel and stores their IDs in the database. When a
// puzzle's status is updated, it handles that also.
func (s *Client) IdempotentCreateUpdate(ctx context.Context, puzzle state.Puzzle) (state.Puzzle, error) {
	// 1. Create the spreadsheet, if required
	if puzzle.SpreadsheetID == "" {
		spreadsheet, err := s.drive.CreateSheet(ctx, puzzle.Name, puzzle.Round.Name)
		if err != nil {
			return puzzle, err
		}

		puzzle, err = s.state.UpdatePuzzle(ctx, puzzle.ID,
			func(puzzle *state.RawPuzzle) error {
				puzzle.SpreadsheetID = spreadsheet
				return nil
			},
		)
		if err != nil {
			return puzzle, err
		}

		err = s.UpdateSpreadsheet(ctx, puzzle)
		if err != nil {
			return puzzle, err
		}
	}

	// 2. Create the Discord channel, if required
	if puzzle.DiscordChannel == "" {
		log.Printf("Adding channel for new puzzle %q", puzzle.Name)
		category, err := s.discordGetOrCreateCategory(puzzle)
		if err != nil {
			return puzzle, err
		}

		channel, err := s.discord.CreateChannel(puzzle.Name, category)
		if err != nil {
			return puzzle, err
		}

		puzzle, err = s.state.UpdatePuzzle(ctx, puzzle.ID,
			func(puzzle *state.RawPuzzle) error {
				puzzle.DiscordChannel = channel.ID
				return nil
			},
		)
		if err != nil {
			return puzzle, err
		}

		err = s.UpdateDiscordPin(puzzle)
		if err != nil {
			return puzzle, err
		}

		if err := s.UpdateDiscordChannel(puzzle); err != nil {
			return puzzle, err
		}

		// Treat Discord channel creation as the sentinel to also notify the
		// team about the new puzzle.
		if err := s.NotifyNewPuzzle(puzzle); err != nil {
			return puzzle, err
		}
	}

	// 3. Update the spreadsheet and Discord channel with new information
	var err error
	puzzle, err = s.HandleStatusChange(ctx, puzzle, false)
	if err != nil {
		return puzzle, err
	}

	return puzzle, nil
}

// HandleStatusChange synchronizes Discord and sends notifications when the
// puzzle status changes. It's called by IdempotentCreateUpdate. If you're
// calling this from a slash command handler, and the is going to acknowledge
// the user in the puzzle channel, you can set `botRequest=true` to suppress
// notifications to the puzzle channel.
func (s *Client) HandleStatusChange(
	ctx context.Context, puzzle state.Puzzle, botRequest bool,
) (state.Puzzle, error) {
	log.Printf("syncer: handling status change for %q", puzzle.Name)
	if puzzle.DiscordChannel == "" {
		return puzzle, xerrors.Errorf("puzzle is a placeholder puzzle, skipping")
	}

	var err error
	err = s.parallelHardUpdate(ctx, puzzle)
	if err != nil {
		return puzzle, err
	}

	// Send notifications
	if puzzle.Status.IsSolved() {
		// Puzzle solved and answer entered! (Suppress puzzle channel
		// notification if this is a bot request, since the bot will also
		// respond in the puzzle channel.)
		err = s.NotifyPuzzleSolved(puzzle, botRequest)
		if err != nil {
			return puzzle, err
		}

		// Also unset the voice room, if applicable
		if puzzle.VoiceRoom != "" {
			puzzle, err = s.state.UpdatePuzzle(ctx, puzzle.ID,
				func(puzzle *state.RawPuzzle) error {
					puzzle.VoiceRoom = ""
					return nil
				},
			)
			if err != nil {
				return puzzle, err
			}
			if err = s.UpdateDiscordPin(puzzle); err != nil {
				return puzzle, err
			}
			if err = s.SyncVoiceRooms(ctx); err != nil {
				return puzzle, err
			}
		}
	} else if puzzle.Status == status.Working {
		if err = s.NotifyPuzzleWorking(puzzle); err != nil {
			return puzzle, err
		}
	}
	return puzzle, nil
}

// ForceUpdate is a big hammer that will update Discord and Google Drive,
// including overwriting the channel name, spreadsheet name, etc. It also
// re-sends any status change notifications.
func (s *Client) ForceUpdate(ctx context.Context, puzzle state.Puzzle) (state.Puzzle, error) {
	if puzzle.SpreadsheetID == "-" || puzzle.DiscordChannel == "-" {
		return puzzle, xerrors.Errorf("puzzle is a placeholder puzzle, skipping")
	}

	var err error
	puzzle, err = s.IdempotentCreateUpdate(ctx, puzzle)
	if err != nil {
		return puzzle, err
	}

	err = s.parallelHardUpdate(ctx, puzzle)
	if err != nil {
		return puzzle, err
	}
	return puzzle, nil
}

// parallelHardUpdate updates the Discord pinned message, the Discord channel
// name/category, and the Google spreadsheet name. It's called when the puzzle
// status changes, and as part of ForceUpdate().
func (s *Client) parallelHardUpdate(ctx context.Context, puzzle state.Puzzle) error {
	var wg sync.WaitGroup
	var ch = make(chan error, 3)
	wg.Add(3)
	go func() { ch <- s.UpdateDiscordPin(puzzle); wg.Done() }()
	go func() { ch <- s.UpdateDiscordChannel(puzzle); wg.Done() }()
	go func() { ch <- s.UpdateSpreadsheet(ctx, puzzle); wg.Done() }()
	wg.Wait()
	close(ch)
	for err := range ch {
		if err != nil {
			return err
		}
	}
	return nil
}
