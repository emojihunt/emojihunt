package syncer

import (
	"context"
	"fmt"
	"log"

	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
)

type Syncer struct {
	airtable *client.Airtable
	discord  *client.Discord
	drive    *client.Drive
}

func New(airtable *client.Airtable, discord *client.Discord, drive *client.Drive) *Syncer {
	return &Syncer{airtable, discord, drive}
}

// IdempotentCreateUpdate synchronizes Discord and Google Drive with the puzzle
// information in Airtable. When a puzzle is newly added, it creates the
// spreadsheet and Discord channel and links them in Airtable. When a puzzle's
// status is updated, it handles that also.
func (s *Syncer) IdempotentCreateUpdate(ctx context.Context, puzzle *schema.Puzzle) (*schema.Puzzle, error) {
	// 1. Create the spreadsheet, if required
	if puzzle.SpreadsheetID == "" {
		spreadsheet, err := s.drive.CreateSheet(ctx, puzzle.Name, puzzle.Round.Name)
		if err != nil {
			return nil, fmt.Errorf("error creating spreadsheet for %q: %v", puzzle.Name, err)
		}

		puzzle, err = s.airtable.UpdateSpreadsheetID(puzzle, spreadsheet)
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

		puzzle, err = s.airtable.UpdateDiscordChannel(puzzle, channel.ID)
		if err != nil {
			return nil, fmt.Errorf("error setting discord channel for puzzle %q: %v", puzzle.Name, err)
		}

		err = s.discordCreateUpdatePin(puzzle)
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
		puzzle, err = s.BasicUpdate(ctx, puzzle, false)
		if err != nil {
			return nil, err
		}
	}

	return puzzle, nil
}

func (s *Syncer) BasicUpdate(ctx context.Context, puzzle *schema.Puzzle, botRequest bool) (*schema.Puzzle, error) {
	var err error
	if err := s.discordCreateUpdatePin(puzzle); err != nil {
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
		puzzle, err = s.airtable.UpdateBotFields(puzzle, puzzle.Status, puzzle.ShouldArchive(), puzzle.Pending)
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
	}
	if err != nil {
		return nil, fmt.Errorf("error posting puzzle status announcement: %v", err)
	}
	return puzzle, nil
}

func (s *Syncer) ForceUpdate(ctx context.Context, puzzle *schema.Puzzle) (*schema.Puzzle, error) {
	var err error
	puzzle, err = s.IdempotentCreateUpdate(ctx, puzzle)
	if err != nil {
		return nil, err
	}

	if err := s.discordCreateUpdatePin(puzzle); err != nil {
		return nil, fmt.Errorf("unable to set puzzle status message for %q: %w", puzzle.Name, err)
	}

	if err := s.discordUpdateChannel(puzzle); err != nil {
		return nil, fmt.Errorf("unable to set channel category for %q: %v", puzzle.Name, err)
	}

	if err := s.driveUpdateSpreadsheet(ctx, puzzle); err != nil {
		return nil, fmt.Errorf("unable to update spreadsheet title and folder for %q: %v", puzzle.Name, err)
	}

	// Update bot status in Airtable and *mark as not pending* if applicable
	puzzle, err = s.airtable.UpdateBotFields(puzzle, puzzle.Status, puzzle.ShouldArchive(), false)
	if err != nil {
		return nil, err
	}
	return puzzle, nil
}
