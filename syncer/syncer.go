package syncer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
)

type Syncer struct {
	airtable *client.Airtable
	discord  *client.Discord
	drive    *client.Drive

	VoiceRoomMutex       sync.Mutex
	DiscordCategoryMutex sync.Mutex
}

func New(airtable *client.Airtable, discord *client.Discord, drive *client.Drive) *Syncer {
	return &Syncer{
		airtable: airtable,
		discord:  discord,
		drive:    drive,
	}
}

// IdempotentCreateUpdate synchronizes Discord and Google Drive with the puzzle
// information in Airtable. When a puzzle is newly added, it creates the
// spreadsheet and Discord channel and links them in Airtable. When a puzzle's
// status is updated, it handles that also.
func (s *Syncer) IdempotentCreateUpdate(ctx context.Context, puzzle *schema.Puzzle) (*schema.Puzzle, error) {
	// 1. Create the spreadsheet, if required
	if puzzle.SpreadsheetID == "" {
		spreadsheet, err := s.drive.CreateSheet(ctx, puzzle.Name, puzzle.Rounds[0].Name)
		if err != nil {
			return nil, fmt.Errorf("error creating spreadsheet for %q: %v", puzzle.Name, err)
		}

		puzzle, err = s.airtable.SetSpreadsheetID(puzzle, spreadsheet)
		if err != nil {
			return nil, fmt.Errorf("error setting spreadsheet id for puzzle %q: %v", puzzle.Name, err)
		}

		err = s.driveUpdateSpreadsheet(ctx, puzzle)
		if err != nil {
			return nil, fmt.Errorf("error setting up spreadsheet for puzzle %q: %v", puzzle.Name, err)
		}
	}

	// 2. Create the Discord channel, if required
	if puzzle.DiscordChannel == "" {
		log.Printf("Adding channel for new puzzle %q", puzzle.Name)
		channel, err := s.discord.CreateChannel(puzzle.Name)
		if err != nil {
			return nil, fmt.Errorf("error creating discord channel for %q: %v", puzzle.Name, err)
		}

		puzzle, err = s.airtable.SetDiscordChannel(puzzle, channel.ID)
		if err != nil {
			return nil, fmt.Errorf("error setting discord channel for puzzle %q: %v", puzzle.Name, err)
		}

		err = s.DiscordCreateUpdatePin(puzzle)
		if err != nil {
			return nil, fmt.Errorf("error pinning info for puzzle %q: %v", puzzle.Name, err)
		}

		if err := s.discordUpdateChannel(puzzle); err != nil {
			return nil, fmt.Errorf("unable to set channel category for %q: %v", puzzle.Name, err)
		}

		// Treat Discord channel creation as the sentinel to also notify the
		// team about the new puzzle.
		if err := s.notifyNewPuzzle(puzzle); err != nil {
			return nil, fmt.Errorf("error notifying channel about new puzzle %q: %v", puzzle.Name, err)
		}
	}

	// 3. Update the spreadsheet and Discord channel with new information, if required
	if puzzle.Status != puzzle.LastBotStatus || puzzle.ShouldArchive() != puzzle.Archived {
		var err error
		puzzle, err = s.HandleStatusChange(ctx, puzzle, false)
		if err != nil {
			return nil, err
		}
	} else if puzzle.LastModifiedBy != s.airtable.BotUserID {
		var err error
		if err = s.DiscordCreateUpdatePin(puzzle); err != nil {
			return nil, fmt.Errorf("unable to set puzzle status message for %q: %w", puzzle.Name, err)
		}
		puzzle, err = s.airtable.SetBotFields(puzzle, puzzle.Status, puzzle.ShouldArchive(), puzzle.Pending)
		if err != nil {
			return nil, fmt.Errorf("failed to update bot fields for puzzle %q: %v", puzzle.Name, err)
		} else if puzzle.LastModifiedBy != s.airtable.BotUserID {
			panic(fmt.Errorf("updated puzzle but user ID does not match bot's, existing to avoid infinite loop: %#v", puzzle))
		}
	}

	return puzzle, nil
}

// HandleStatusChange synchronizes Discord and sends notifications when the
// puzzle status changes. It's called by IdempotentCreateUpdate. If you're
// calling this from a slash command handler, and the is going to acknowledge
// the user in the puzzle channel, you can set `botRequest=true` to suppress
// notifications to the puzzle channel.
func (s *Syncer) HandleStatusChange(ctx context.Context, puzzle *schema.Puzzle, botRequest bool) (*schema.Puzzle, error) {
	log.Printf("syncer: handling status change for %q", puzzle.Name)

	var err error
	if err := s.DiscordCreateUpdatePin(puzzle); err != nil {
		return nil, fmt.Errorf("unable to set puzzle status message for %q: %w", puzzle.Name, err)
	}

	if err := s.discordUpdateChannel(puzzle); err != nil {
		return nil, fmt.Errorf("unable to set channel category for %q: %v", puzzle.Name, err)
	}

	if err := s.driveUpdateSpreadsheet(ctx, puzzle); err != nil {
		return nil, fmt.Errorf("unable to update spreadsheet title and folder for %q: %v", puzzle.Name, err)
	}

	// Update bot status in Airtable, unless we're in a bot handler and this has
	// already been done.
	if !botRequest {
		var err error
		puzzle, err = s.airtable.SetBotFields(puzzle, puzzle.Status, puzzle.ShouldArchive(), puzzle.Pending)
		if err != nil {
			return nil, fmt.Errorf("failed to update bot fields for puzzle %q: %v", puzzle.Name, err)
		}
	}

	// Send notifications
	if puzzle.Status.IsSolved() {
		if puzzle.Answer != "" {
			// Puzzle solved and answer entered! (Suppress puzzle channel
			// notification if this is a bot request, since the bot will also
			// respond in the puzzle channel.)
			err = s.notifyPuzzleFullySolved(puzzle, botRequest)
		} else {
			// Puzzle marked as solved but answer needs to be entered in
			// Airtable...
			err = s.notifyPuzzleSolvedMissingAnswer(puzzle)
		}
		if err != nil {
			return nil, fmt.Errorf("error posting puzzle status announcement: %v", err)
		}

		// Also unset the voice room, if applicable
		if puzzle.VoiceRoom != "" {
			s.VoiceRoomMutex.Lock()
			defer s.VoiceRoomMutex.Unlock()
			puzzle, err = s.airtable.SetVoiceRoom(puzzle, nil)
			if err != nil {
				return nil, fmt.Errorf("error unsetting voice room: %v", err)
			}
			if err = s.DiscordCreateUpdatePin(puzzle); err != nil {
				return nil, err
			}
			if err = s.SyncVoiceRooms(ctx); err != nil {
				return nil, err
			}
		}
	} else if puzzle.Status == schema.Working {
		if err = s.notifyPuzzleWorking(puzzle); err != nil {
			return nil, err
		}
	}
	return puzzle, nil
}

// ForceUpdate is a big hammer that will update Discord and Google Drive,
// including overwriting the channel name, spreadsheet name, etc. It also
// re-sends any status change notifications.
func (s *Syncer) ForceUpdate(ctx context.Context, puzzle *schema.Puzzle) (*schema.Puzzle, error) {
	if len(puzzle.SpreadsheetID) < 1 || len(puzzle.DiscordChannel) < 1 {
		return nil, errors.New("puzzle is missing a spreadsheet or Discord channel, skipping")
	}

	var err error
	puzzle, err = s.IdempotentCreateUpdate(ctx, puzzle)
	if err != nil {
		return nil, err
	}

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
			return nil, err
		}
	}

	// Update bot status in Airtable and *mark as not pending* if applicable
	puzzle, err = s.airtable.SetBotFields(puzzle, puzzle.Status, puzzle.ShouldArchive(), false)
	if err != nil {
		return nil, err
	}
	return puzzle, nil
}
