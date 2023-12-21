package discovery

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/db"
	"golang.org/x/xerrors"
)

func (d *Poller) SyncPuzzles(ctx context.Context, puzzles []db.DiscoveredPuzzle) error {
	puzzleMap := make(map[string]db.DiscoveredPuzzle)
	for _, puzzle := range puzzles {
		puzzleMap[puzzle.URL] = puzzle
	}

	// Filter out known puzzles; add remaining puzzles
	fragments, rounds, err := d.db.ListPuzzleFragmentsAndRounds(ctx)
	if err != nil {
		return err
	}

	var newPuzzles []db.NewPuzzle
	newRounds := make(map[string][]db.DiscoveredPuzzle)
	for _, puzzle := range puzzleMap {
		if fragments[strings.ToUpper(puzzle.URL)] ||
			fragments[strings.ToUpper(puzzle.Name)] {
			// skip if name or URL matches an existing puzzle
			continue
		}
		if round, ok := rounds[puzzle.RoundName]; ok {
			log.Printf("discovery: preparing to add puzzle %q (%s) in round %q",
				puzzle.Name, puzzle.URL, puzzle.RoundName)
			newPuzzles = append(newPuzzles, db.NewPuzzle{
				Name:  puzzle.Name,
				Round: round,
				URL:   puzzle.URL,
			})
		} else {
			// puzzle belongs to a new round
			newRounds[puzzle.RoundName] = append(newRounds[puzzle.RoundName], puzzle)
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
	for _, puzzle := range newPuzzles {
		msg += fmt.Sprintf("%s %s\n%s\n\n", puzzle.Round.Emoji, puzzle.Name, puzzle.URL)
	}
	msg += "Reminder: use `/huntbot kill` to stop the bot.\n```\n"
	_, err := d.discord.ChannelSend(d.discord.QMChannel, msg)
	if err != nil {
		return err
	}
	return d.createPuzzles(ctx, newPuzzles)
}

func (d *Poller) handleNewRounds(ctx context.Context, newRounds map[string][]db.DiscoveredPuzzle) error {
	d.state.Lock()
	defer d.state.CommitAndUnlock(ctx)

	var errs []error
	for name, puzzles := range newRounds {
		if _, ok := d.state.DiscoveryNewRounds[name]; ok {
			continue
		}

		msg := fmt.Sprintf("```*** ❓ NEW ROUND: \"%s\" ***\n\n", name)
		for _, puzzle := range puzzles {
			msg += fmt.Sprintf("%s\n%s\n\n", puzzle.Name, puzzle.URL)
		}
		msg += "Reminder: use `/huntbot kill` to stop the bot.\n\n"
		msg += ">> REACT TO PROPOSE AN EMOJI FOR THIS ROUND <<\n```\n"

		id, err := d.discord.ChannelSend(d.discord.QMChannel, msg)
		if err != nil {
			errs = append(errs, err)
		} else {
			d.state.DiscoveryNewRounds[name] = db.DiscoveredRound{
				MessageID: id,
				Name:      name,
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

func (d *Poller) createRound(ctx context.Context, name string, roundInfo db.DiscoveredRound) error {
	emoji, err := d.getTopReaction(roundInfo.MessageID)
	if err != nil {
		return err
	} else if emoji == "" {
		return xerrors.Errorf("no reaction for message")
	}

	if d.state.IsKilled() {
		return xerrors.Errorf("huntbot is disabled")
	}
	round, err := d.db.CreateRound(ctx, db.Round{
		Name:  roundInfo.Name,
		Emoji: emoji,
	})
	if err != nil {
		return err
	}

	var puzzles = make([]db.NewPuzzle, len(roundInfo.Puzzles))
	for i, puzzle := range roundInfo.Puzzles {
		puzzles[i] = db.NewPuzzle{
			Name:  puzzle.Name,
			Round: round,
			URL:   puzzle.URL,
		}
	}
	return d.createPuzzles(ctx, puzzles)
}
