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

func (c *Client) TriggerPuzzle(ctx context.Context, previous *state.Puzzle, puzzle state.Puzzle) error {
	var err error
	if puzzle.SpreadsheetID == "" {
		puzzle, err = c.CreateSpreadsheet(ctx, puzzle)
		if err != nil {
			return err
		}
	}
	if puzzle.DiscordChannel == "" {
		puzzle, err = c.CreateDiscordChannel(ctx, puzzle)
		if err != nil {
			return err
		}
	}
	if previous == nil {
		if err := c.NotifyNewPuzzle(puzzle); err != nil {
			return err
		}
	}
	// TODO: ...
	return nil
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
