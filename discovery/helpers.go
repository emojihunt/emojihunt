package discovery

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/state"
	"golang.org/x/xerrors"
)

func (c *Client) handleNewPuzzles(ctx context.Context, puzzles []state.RawPuzzle) error {
	msg := "```\n*** 🧐 NEW PUZZLES ***\n\n"
	for _, puzzle := range puzzles {
		round, err := c.state.GetRound(ctx, puzzle.Round)
		if err != nil {
			return err
		}
		msg += fmt.Sprintf("%s %s\n%s\n\n", round.Emoji, puzzle.Name, puzzle.PuzzleURL)
	}
	msg += "Reminder: use `/qm discovery pause` to stop the bot.\n```\n"
	_, err := c.discord.ChannelSend(c.discord.QMChannel, msg)
	if err != nil {
		return err
	}
	return c.createPuzzles(ctx, puzzles)
}

func (c *Client) handleNewRounds(ctx context.Context, rounds map[string][]state.DiscoveredPuzzle) error {
	previouslyDiscovered, err := c.state.DiscoveredRounds(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for name, puzzles := range rounds {
		if _, ok := previouslyDiscovered[name]; ok {
			log.Printf("discovery: skipping previously-found round %q", name)
			continue
		}
		log.Printf("discovery: found new round %q", name)

		msg := fmt.Sprintf("```*** ❓ NEW ROUND: \"%s\" ***\n\n", name)
		for _, puzzle := range puzzles {
			msg += fmt.Sprintf("%s\n%s\n\n", puzzle.Name, puzzle.URL)
		}
		msg += "Reminder: use `/qm discovery pause` to stop the bot.\n\n"
		msg += ">> REACT TO PROPOSE AN EMOJI FOR THIS ROUND <<\n```\n"

		id, err := c.discord.ChannelSend(c.discord.QMChannel, msg)
		if err != nil {
			errs = append(errs, err)
		} else {
			var round = state.DiscoveredRound{
				MessageID:  id,
				Name:       name,
				NotifiedAt: time.Now(),
				Puzzles:    puzzles,
			}
			previouslyDiscovered[name] = round
			c.rounds <- round
		}

		if err := ctx.Err(); err != nil {
			if len(errs) > 0 {
				break
			} else {
				return err
			}
		}
	}
	if len(errs) > 0 {
		return xerrors.Errorf("errors sending new round notifications: %#v", spew.Sdump(errs))
	}
	return nil
}

func (c *Client) createPuzzles(ctx context.Context, puzzles []state.RawPuzzle) error {
	for _, puzzle := range puzzles {
		// Pause briefly to allow QMs time to cancel...
		select {
		case <-ctx.Done():
			return xerrors.Errorf("createPuzzles: %w", ctx.Err())
		case <-time.After(1 * time.Second):
		}

		round, err := c.state.GetRound(ctx, puzzle.Round)
		if err != nil {
			return err
		}
		puzzle.SpreadsheetID = ""
		puzzle.SpreadsheetID, err = c.sync.CreateSpreadsheet(ctx, puzzle)
		if err != nil {
			return err
		}
		puzzle.DiscordChannel = ""
		puzzle.DiscordChannel, err = c.sync.CreateDiscordChannel(ctx, puzzle, round)
		if err != nil {
			return err
		}

		_, _, err = c.state.CreatePuzzle(ctx, puzzle)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) createRound(ctx context.Context,
	round state.DiscoveredRound, emoji string) error {

	var err error
	var dbRound = state.Round{Name: round.Name, Emoji: emoji}
	dbRound.DriveFolder, err = c.sync.CreateDriveFolder(ctx, dbRound)
	if err != nil {
		return err
	}
	created, _, err := c.state.CreateRound(ctx, dbRound)
	if err != nil {
		return err
	}

	var puzzles = make([]state.RawPuzzle, len(round.Puzzles))
	for i, puzzle := range round.Puzzles {
		puzzles[i] = state.RawPuzzle{
			Name:      puzzle.Name,
			Round:     created.ID,
			PuzzleURL: puzzle.URL,
		}
	}
	return c.createPuzzles(ctx, puzzles)
}
