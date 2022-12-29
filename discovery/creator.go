package discovery

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/schema"
)

func (d *Poller) SyncPuzzles(ctx context.Context, puzzles []*DiscoveredPuzzle) error {
	puzzleMap := make(map[string]*DiscoveredPuzzle)
	for _, puzzle := range puzzles {
		puzzleMap[puzzle.URL.String()] = puzzle
	}

	// Filter out known puzzles; add remaining puzzles
	fragments, rounds, err := d.airtable.ListPuzzleFragmentsAndRounds()
	if err != nil {
		return err
	}

	var newPuzzles []schema.NewPuzzle
	newRounds := make(map[string][]*DiscoveredPuzzle)
	for _, puzzle := range puzzleMap {
		if fragments[strings.ToUpper(puzzle.URL.String())] ||
			fragments[strings.ToUpper(puzzle.Name)] {
			// skip if name or URL matches an existing puzzle
			continue
		}
		round, ok := rounds[puzzle.Round]
		if !ok {
			newRounds[puzzle.Round] = append(newRounds[puzzle.Round], puzzle)
			continue
		}
		log.Printf("discovery: preparing to add puzzle %q (%s) in round %q", puzzle.Name, puzzle.URL.String(), puzzle.Round)
		newPuzzles = append(newPuzzles, schema.NewPuzzle{
			Name:      puzzle.Name,
			Round:     round,
			PuzzleURL: puzzle.URL.String(),
		})
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

func (d *Poller) handleNewPuzzles(ctx context.Context, newPuzzles []schema.NewPuzzle) error {
	msg := "```\n*** 🧐 NEW PUZZLES ***\n\n"
	for i, puzzle := range newPuzzles {
		if i == newPuzzleLimit {
			msg += fmt.Sprintf("(...and more, %d in total...)\n\n", len(newPuzzles))
			break
		}
		msg += fmt.Sprintf("%s %s\n%s\n\n", puzzle.Round.Emoji, puzzle.Name, puzzle.PuzzleURL)
	}

	var paused bool
	if len(newPuzzles) > newPuzzleLimit {
		paused = true
		msg += fmt.Sprintf(
			"💥 Too many new puzzles! Puzzle creation paused, please contact #%s.\n",
			d.discord.TechChannel.Name,
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

	// Warning! Puzzle locks are acquired here and must be released before this
	// function returns.
	created, err := d.airtable.AddPuzzles(newPuzzles)
	if err != nil {
		return err
	}

	time.Sleep(puzzleCreationPause)

	var errs []error
	for _, puzzle := range created {
		if d.state.IsKilled() {
			errs = append(errs, fmt.Errorf("huntbot is disabled"))
		} else {
			if _, err := d.syncer.ForceUpdate(ctx, &puzzle); err != nil {
				errs = append(errs, err)
			}
		}
		puzzle.Unlock()
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors sending new puzzle notifications: %#v", spew.Sdump(errs))
	}
	return nil
}

func (d *Poller) handleNewRounds(ctx context.Context, newRounds map[string][]*DiscoveredPuzzle) error {
	if len(newRounds) > newRoundLimit {
		msg := fmt.Sprintf(
			"```💥 Too many new rounds! Round creation paused, please contact #%s.\n```\n",
			d.discord.TechChannel.Name,
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
			d.state.DiscoveryNewRounds[name] = id
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors sending new round notifications: %#v", spew.Sdump(errs))
	}
	return nil
}
