package discovery

import (
	"time"

	"github.com/emojihunt/emojihunt/state"
	"github.com/getsentry/sentry-go"
	"golang.org/x/net/context"
)

func (p *Poller) RoundCreationWorker(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", "discovery.rounds")
	})
	ctx = sentry.SetHubOnContext(ctx, hub)
	// *do* allow panics to bubble up to main()

	rounds, err := p.state.DiscoveredRounds(ctx)
	if err != nil {
		panic(err)
	}

	var wakeup = time.Now()
	for {
		select {
		case v, ok := <-p.roundCh:
			// Read and record one round from the queue
			if !ok {
				panic("channel closed")
			}
			rounds[v.Name] = v
			if err := p.state.SetDiscoveredRounds(ctx, rounds); err != nil {
				panic(err)
			}
			continue
		case <-time.After(time.Until(wakeup)):
			// Periodically process any round(s) that have passed the timeout
		case <-ctx.Done():
			return
		}

		// Process any round(s) that have passed the timeout
		if p.state.IsEnabled(ctx) {
			var queue []state.DiscoveredRound
			for _, round := range rounds {
				if wakeup.After(round.NotifiedAt.Add(roundCreationPause)) {
					queue = append(queue, round)
				}
			}
			for _, round := range queue {
				if emoji, err := p.getTopReaction(round.MessageID); err != nil {
					sentry.GetHubFromContext(ctx).CaptureException(err)
				} else if emoji == "" {
					// QM hasn't assigned an emoji yet
					continue
				} else if err := p.createRound(ctx, round, emoji); err != nil {
					sentry.GetHubFromContext(ctx).CaptureException(err)
				} else {
					// Success!
					delete(rounds, round.Name)
					break // only process one round at a time
				}
			}
			if err := p.state.SetDiscoveredRounds(ctx, rounds); err != nil {
				panic(err)
			}
		}
		wakeup = time.Now().Add(roundCreationPause)
	}
}

func (p *Poller) getTopReaction(messageID string) (string, error) {
	msg, err := p.discord.GetMessage(p.discord.QMChannel, messageID)
	if err != nil {
		return "", err
	}

	emoji, count := "", 0
	for _, reaction := range msg.Reactions {
		if reaction.Count > count && reaction.Emoji.Name != "" {
			emoji = reaction.Emoji.Name
			count = reaction.Count
		}
	}
	return emoji, nil
}

func (p *Poller) createRound(ctx context.Context, round state.DiscoveredRound,
	emoji string) error {

	created, err := p.state.CreateRound(ctx, state.Round{
		Name:  round.Name,
		Emoji: emoji,
	})
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
	return p.createPuzzles(ctx, puzzles)
}
