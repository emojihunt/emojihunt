package discovery

import (
	"context"
	"log"
	"strings"
	m "sync"
	"time"

	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/sync"
	"github.com/getsentry/sentry-go"
)

type Client struct {
	discord *discord.Client
	state   *state.Client
	sync    *sync.Client
	rounds  chan state.DiscoveredRound

	mutex m.Mutex
}

func New(discord *discord.Client, s *state.Client, y *sync.Client) *Client {
	return &Client{discord, s, y, make(chan state.DiscoveredRound), m.Mutex{}}
}

func (c *Client) Watch(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", "discovery.watch")
	})
	ctx = sentry.SetHubOnContext(ctx, hub)
	// *do* allow panics to bubble up to main()

	for {
		ctx, cancel := context.WithCancel(ctx)
		if !c.state.IsEnabled(ctx) {
			log.Printf("discovery: disabled via kill switch")
		} else {
			err := func() error {
				config, err := c.state.DiscoveryConfig(ctx)
				if err != nil {
					return err
				}
				if config.PuzzlesURL == "" {
					log.Printf("discovery: disabled because puzzle_url is blank")
					return nil // disabled
				}
				poller, err := NewPoller(config)
				if err != nil {
					return err
				}
				go poller.Poll(ctx, c)
				return nil
			}()
			if err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
			}
		}

		select {
		case <-ctx.Done():
			log.Printf("discovery: exiting worker!")
			cancel()
			return
		case <-c.sync.RestartDiscovery:
			log.Printf("discovery: restarting...")
			cancel()
		}
	}
}

func (c *Client) SyncPuzzles(ctx context.Context, puzzles []state.DiscoveredPuzzle) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	log.Printf("discovery: syncing %d puzzles", len(puzzles))

	// Filter out known puzzles; add remaining puzzles
	var fragments = make(map[string]bool)
	existing, err := c.state.ListPuzzles(ctx)
	if err != nil {
		return err
	}
	for _, puzzle := range existing {
		fragments[strings.ToUpper(puzzle.Name)] = true
		fragments[strings.ToUpper(puzzle.PuzzleURL)] = true
	}

	var roundsByName = make(map[string]state.Round)
	rounds, err := c.state.ListRounds(ctx)
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
		err := c.handleNewPuzzles(ctx, newPuzzles)
		if err != nil {
			return err
		}
	}

	if len(newRounds) > 0 {
		err := c.handleNewRounds(ctx, newRounds)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *Client) RoundCreationWorker(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", "discovery.rounds")
	})
	ctx = sentry.SetHubOnContext(ctx, hub)
	// *do* allow panics to bubble up to main()

	rounds, err := c.state.DiscoveredRounds(ctx)
	if err != nil {
		panic(err)
	}

	var wakeup = time.Now()
	for {
		select {
		case v, ok := <-c.rounds:
			// Read and record one round from the queue
			if !ok {
				panic("channel closed")
			}
			rounds[v.Name] = v
			if err := c.state.SetDiscoveredRounds(ctx, rounds); err != nil {
				panic(err)
			}
			continue
		case <-time.After(time.Until(wakeup)):
			// Periodically process any round(s) that have passed the timeout
		case <-ctx.Done():
			return
		}

		// Process any round(s) that have passed the timeout
		var queue []state.DiscoveredRound
		for _, round := range rounds {
			if wakeup.After(round.NotifiedAt.Add(roundCreationPause)) {
				queue = append(queue, round)
			}
		}
		for _, round := range queue {
			if emoji, err := c.discord.GetTopReaction(c.discord.QMChannel, round.MessageID); err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
			} else if emoji == "" {
				// QM hasn't assigned an emoji yet
				continue
			} else if err := c.createRound(ctx, round, emoji); err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
			} else {
				// Success!
				delete(rounds, round.Name)
				break // only process one round at a time
			}
		}
		if err := c.state.SetDiscoveredRounds(ctx, rounds); err != nil {
			panic(err)
		}
		wakeup = time.Now().Add(roundCreationPause)
	}
}
