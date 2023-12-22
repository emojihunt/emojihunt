package discovery

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/state"
	"golang.org/x/xerrors"
)

func (p *Poller) SyncPuzzles(ctx context.Context, puzzles []state.DiscoveredPuzzle) error {
	// Filter out known puzzles; add remaining puzzles
	var fragments = make(map[string]bool)
	existing, err := p.state.ListPuzzles(ctx)
	if err != nil {
		return err
	}
	for _, puzzle := range existing {
		fragments[strings.ToUpper(puzzle.Name)] = true
		fragments[strings.ToUpper(puzzle.PuzzleURL)] = true
	}

	var roundsByName = make(map[string]state.Round)
	rounds, err := p.state.ListRounds(ctx)
	if err != nil {
		return err
	}
	for _, round := range rounds {
		roundsByName[strings.ToUpper(round.Name)] = round
	}

	var newPuzzles []state.RawPuzzle
	newRounds := make(map[string][]state.DiscoveredPuzzle)
	for _, puzzle := range puzzles {
		if fragments[strings.ToUpper(puzzle.URL)] ||
			fragments[strings.ToUpper(puzzle.Name)] {
			// skip if name or URL matches an existing puzzle
			continue
		}
		if round, ok := roundsByName[strings.ToUpper(puzzle.RoundName)]; ok {
			log.Printf("discovery: preparing to add puzzle %q (%s) in round %q",
				puzzle.Name, puzzle.URL, puzzle.RoundName)
			newPuzzles = append(newPuzzles, state.RawPuzzle{
				Name:      puzzle.Name,
				Round:     round.ID,
				PuzzleURL: puzzle.URL,
			})
		} else {
			// puzzle belongs to a new round
			newRounds[puzzle.RoundName] = append(newRounds[puzzle.RoundName], puzzle)
		}
	}

	if len(newPuzzles) > 0 {
		err := p.handleNewPuzzles(ctx, newPuzzles)
		if err != nil {
			return err
		}
	}

	if len(newRounds) > 0 {
		err := p.handleNewRounds(ctx, newRounds)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Poller) handleNewPuzzles(ctx context.Context, newPuzzles []state.RawPuzzle) error {
	msg := "```\n*** 🧐 NEW PUZZLES ***\n\n"
	for _, puzzle := range newPuzzles {
		round, err := p.state.GetRound(ctx, puzzle.Round)
		if err != nil {
			return err
		}
		msg += fmt.Sprintf("%s %s\n%s\n\n", round.Emoji, puzzle.Name, puzzle.PuzzleURL)
	}
	msg += "Reminder: use `/huntbot kill` to stop the bot.\n```\n"
	_, err := p.discord.ChannelSend(p.discord.QMChannel, msg)
	if err != nil {
		return err
	}
	return p.createPuzzles(ctx, newPuzzles)
}

func (p *Poller) handleNewRounds(ctx context.Context, newRounds map[string][]state.DiscoveredPuzzle) error {
	previouslyDiscovered, err := p.state.DiscoveredRounds(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for name, puzzles := range newRounds {
		if _, ok := previouslyDiscovered[name]; ok {
			continue
		}

		msg := fmt.Sprintf("```*** ❓ NEW ROUND: \"%s\" ***\n\n", name)
		for _, puzzle := range puzzles {
			msg += fmt.Sprintf("%s\n%s\n\n", puzzle.Name, puzzle.URL)
		}
		msg += "Reminder: use `/huntbot kill` to stop the bot.\n\n"
		msg += ">> REACT TO PROPOSE AN EMOJI FOR THIS ROUND <<\n```\n"

		id, err := p.discord.ChannelSend(p.discord.QMChannel, msg)
		if err != nil {
			errs = append(errs, err)
		} else {
			// TODO
			previouslyDiscovered[name] = state.DiscoveredRound{
				MessageID:  id,
				Name:       name,
				NotifiedAt: time.Now(),
				Puzzles:    puzzles,
			}
		}
	}
	if len(errs) > 0 {
		return xerrors.Errorf("errors sending new round notifications: %#v", spew.Sdump(errs))
	}
	return nil
}

func (p *Poller) createPuzzles(ctx context.Context, newPuzzles []state.RawPuzzle) error {
	for _, puzzle := range newPuzzles {
		if !p.state.IsEnabled(ctx) {
			return xerrors.Errorf("huntbot is disabled")
		}
		created, err := p.state.CreatePuzzle(ctx, puzzle)
		if err != nil {
			return err
		}
		_, err = p.syncer.ForceUpdate(ctx, created)
		if err != nil {
			return err
		}
	}
	return nil
}
