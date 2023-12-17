package discovery

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/state"
	"golang.org/x/xerrors"
)

func (d *Poller) SyncPuzzles(ctx context.Context, puzzles []state.DiscoveredPuzzle) error {
	puzzleMap := make(map[string]state.DiscoveredPuzzle)
	for _, puzzle := range puzzles {
		puzzleMap[puzzle.URL] = puzzle
	}

	// Filter out known puzzles; add remaining puzzles
	fragments, rounds, err := d.db.ListPuzzleFragmentsAndRounds(ctx)
	if err != nil {
		return err
	}

	var newPuzzles []db.NewPuzzle
	newRounds := make(map[string][]state.DiscoveredPuzzle)
	for _, puzzle := range puzzleMap {
		if fragments[strings.ToUpper(puzzle.URL)] ||
			fragments[strings.ToUpper(puzzle.Name)] {
			// skip if name or URL matches an existing puzzle
			continue
		}
		if round, ok := rounds[puzzle.Round]; ok {
			log.Printf("discovery: preparing to add puzzle %q (%s) in round %q",
				puzzle.Name, puzzle.URL, puzzle.Round)
			newPuzzles = append(newPuzzles, db.NewPuzzle{
				Name:  puzzle.Name,
				Round: round,
				URL:   puzzle.URL,
			})
		} else {
			// puzzle belongs to a new round
			newRounds[puzzle.Round] = append(newRounds[puzzle.Round], puzzle)
		}
	}

	if len(newPuzzles) > 0 {
		err := d.handleNewPuzzles(ctx, newPuzzles)
		if err != nil {
			return err
		}
	}

	if len(newRounds) > 0 {
		err := d.handleNewRounds(ctx, newRounds)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *Poller) handleNewPuzzles(ctx context.Context, newPuzzles []db.NewPuzzle) error {
	msg := "```\n*** 🧐 NEW PUZZLES ***\n\n"
	for i, puzzle := range newPuzzles {
		if i == newPuzzleLimit {
			msg += fmt.Sprintf("(...and more, %d in total...)\n\n", len(newPuzzles))
			break
		}
		msg += fmt.Sprintf("%s %s\n%s\n\n", "TODO: puzzle.Round.Emoji", puzzle.Name, puzzle.URL)
	}

	var paused bool
	if len(newPuzzles) > newPuzzleLimit {
		paused = true
		msg += fmt.Sprintf(
			"💥 Too many new puzzles! Puzzle creation paused, please contact Tech.\n",
		)
	} else {
		msg += "Reminder: use `/huntbot kill` to stop the bot.\n"
	}
	msg += "```\n"

	_, err := d.discord.ChannelSend(d.discord.QMChannel, msg)
	if err != nil {
		return err
	} else if paused {
		return nil
	}

	return d.createPuzzles(ctx, newPuzzles)
}

func (d *Poller) handleNewRounds(ctx context.Context, newRounds map[string][]state.DiscoveredPuzzle) error {
	if len(newRounds) > newRoundLimit {
		msg := fmt.Sprintf(
			"```💥 Too many new rounds! Round creation paused, please contact Tech.\n```\n",
		)
		_, err := d.discord.ChannelSend(d.discord.QMChannel, msg)
		return err
	}

	d.state.Lock()
	defer d.state.CommitAndUnlock()

	var errs []error
	for name, puzzles := range newRounds {
		if _, ok := d.state.DiscoveryNewRounds[name]; ok {
			continue
		}

		msg := fmt.Sprintf("```*** ❓ NEW ROUND: \"%s\" ***\n\n", name)
		for i, puzzle := range puzzles {
			if i == newPuzzleLimit {
				msg += fmt.Sprintf("(...and more, %d in total...)\n\n", len(puzzles))
				break
			}
			msg += fmt.Sprintf("%s\n%s\n\n", puzzle.Name, puzzle.URL)
		}
		msg += "Reminder: use `/huntbot kill` to stop the bot.\n\n"
		msg += ">> REACT TO PROPOSE AN EMOJI FOR THIS ROUND <<\n```\n"

		id, err := d.discord.ChannelSend(d.discord.QMChannel, msg)
		if err != nil {
			errs = append(errs, err)
		} else {
			d.state.DiscoveryNewRounds[name] = state.NewRound{
				MessageID: id,
				Puzzles:   puzzles,
			}
		}
	}
	if len(errs) > 0 {
		return xerrors.Errorf("errors sending new round notifications: %#v", spew.Sdump(errs))
	}
	return nil
}

func (d *Poller) createPuzzles(ctx context.Context, newPuzzles []db.NewPuzzle) error {
	created, err := d.db.AddPuzzles(ctx, newPuzzles)
	if err != nil {
		return err
	}

	// TODO: handle this...
	// if !newRound {
	// 	time.Sleep(puzzleCreationPause)
	// }

	var errs []error
	for _, puzzle := range created {
		if d.state.IsKilled() {
			errs = append(errs, xerrors.Errorf("huntbot is disabled"))
		} else {
			if _, err := d.syncer.ForceUpdate(ctx, &puzzle); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return xerrors.Errorf("errors sending new puzzle notifications: %#v", spew.Sdump(errs))
	}
	return nil
}

func (d *Poller) createRound(ctx context.Context, name string, roundInfo state.NewRound) error {
	emoji, err := d.getTopReaction(roundInfo.MessageID)
	if err != nil {
		return err
	} else if emoji == "" {
		return xerrors.Errorf("no reaction for message")
	}

	var round = db.Round{ID: 0, Name: "TODO", Emoji: "TODO"}

	var puzzles = make([]db.NewPuzzle, len(roundInfo.Puzzles))
	for i, puzzle := range roundInfo.Puzzles {
		puzzles[i] = db.NewPuzzle{
			Name:  puzzle.Name,
			Round: round.ID,
			URL:   puzzle.URL,
		}
	}
	return d.createPuzzles(ctx, puzzles)
}
