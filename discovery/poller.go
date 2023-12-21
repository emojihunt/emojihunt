package discovery

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/andybalholm/cascadia"
	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
	"github.com/getsentry/sentry-go"
	"golang.org/x/net/websocket"
	"golang.org/x/time/rate"
)

type DiscoveryConfig struct {
	// URL of the "All Puzzles" page on the hunt website
	PuzzlesURL  string `json:"puzzles_url"`
	CookieName  string `json:"cookie_name"`
	CookieValue string `json:"cookie_value"`

	// Group Mode: in many years (2021, 2020, etc.), the puzzle list is grouped
	// by round, and there is some grouping element (e.g. a <section>) for each
	// round that contains both the round name and the list of puzzles.
	//
	// In other years (2022), the puzzle list is presented as a sequence of
	// alternating round names (e.g. <h2>) and puzzle lists (e.g. <table>) with
	// no grouping element. If this is the case, set `groupedMode=false` and use
	// the group selector to select the overall container. Note that the round
	// name element must be an *immediate* child of the container, and the
	// puzzle list element must be its immediate sibling.
	//
	// EXAMPLES
	//
	// 2022 (https://puzzles.mit.edu/2022/puzzles/)
	// - Group:       `section#main-content` (group mode off)
	// - Round Name:  `h2`
	// - Puzzle List: `table`
	//
	// 2021 (https://puzzles.mit.edu/2021/puzzles.html)
	// - Group:       `.info div section` (group mode on)
	// - Round Name:  `a h3`
	// - Puzzle List: `table`
	//
	// 2020 (https://puzzles.mit.edu/2020/puzzles/)
	// - Group:       `#loplist > li:not(:first-child)` (group mode on)
	// - Round Name:  `a`
	// - Puzzle List: `ul li a`
	//
	// 2019 (https://puzzles.mit.edu/2019/puzzle.html)
	// - Group:       `.puzzle-list-section:nth-child(2) .round-list-item` (group mode on)
	// - Round Name:  `.round-list-header`
	// - Puzzle List: `.round-list-item`
	// - Puzzle Item: `.puzzle-list-item a`
	//
	GroupMode          bool   `json:"group_mode"`
	GroupSelector      string `json:"group_selector"`
	RoundNameSelector  string `json:"round_name_selector"`
	PuzzleListSelector string `json:"puzzle_list_selector"`

	// Optional: defaults to "a" (this is probably what you want)
	PuzzleItemSelector string `json:"puzzle_item_selector"`

	// URL of the websocket endpoint (optional)
	WebsocketURL string `json:"websocket_url"`

	// Token to send in the AUTH message (optional)
	WebsocketToken string `json:"websocket_token"`
}

type Poller struct {
	puzzlesURL *url.URL
	cookie     *http.Cookie

	groupMode          bool
	groupSelector      cascadia.Selector
	roundNameSelector  cascadia.Selector
	puzzleListSelector cascadia.Selector
	puzzleItemSelector cascadia.Selector

	wsURL   *url.URL
	wsToken string

	main      context.Context
	discord   *discord.Client
	state     *state.Client
	syncer    *syncer.Syncer
	roundCh   chan state.DiscoveredRound
	wsLimiter *rate.Limiter
}

const (
	pollInterval       = 20 * time.Second
	pollTimeout        = 90 * time.Second
	roundCreationPause = 10 * time.Second
	websocketBurst     = 3
)

var websocketRate = rate.Every(1 * time.Minute)

func New(main context.Context, discord *discord.Client, st *state.Client,
	syncer *syncer.Syncer, config *DiscoveryConfig) *Poller {

	puzzlesURL, err := url.Parse(config.PuzzlesURL)
	if err != nil {
		panic(err)
	}

	var wsURL *url.URL
	if config.WebsocketURL != "" {
		wsURL, err = url.Parse(config.WebsocketURL)
		if err != nil {
			panic(err)
		}
	}

	itemSelector := config.PuzzleItemSelector
	if itemSelector == "" {
		itemSelector = "a"
	}

	return &Poller{
		puzzlesURL: puzzlesURL,
		cookie: &http.Cookie{
			Name:   config.CookieName,
			Value:  config.CookieValue,
			MaxAge: 0,
		},

		groupMode:          config.GroupMode,
		groupSelector:      cascadia.MustCompile(config.GroupSelector),
		roundNameSelector:  cascadia.MustCompile(config.RoundNameSelector),
		puzzleListSelector: cascadia.MustCompile(config.PuzzleListSelector),
		puzzleItemSelector: cascadia.MustCompile(itemSelector),

		wsURL:   wsURL,
		wsToken: config.WebsocketToken,

		main:      main,
		state:     st,
		discord:   discord,
		syncer:    syncer,
		roundCh:   make(chan state.DiscoveredRound),
		wsLimiter: rate.NewLimiter(websocketRate, websocketBurst),
	}
}

func (p *Poller) Poll(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", "discovery")
	})
	ctx = sentry.SetHubOnContext(ctx, hub)
	// *do* allow panics to bubble up to main()

	go p.RoundCreationWorker(ctx)

reconnect:
	for {
		ch, err := p.openWebsocket(p.main)
		if err != nil {
			log.Printf("discovery: failed to open websocket: %v", spew.Sprint(err))
		}

		for {
			if !p.state.IsDisabled(ctx) {
				ctx, cancel := context.WithTimeout(ctx, pollTimeout)
				defer cancel()

				if puzzles, err := p.Scrape(ctx); err != nil {
					hub.CaptureException(err)
				} else if err := p.SyncPuzzles(ctx, puzzles); err != nil {
					hub.CaptureException(err)
				}
			}

			select {
			case <-ctx.Done():
				return
			case _, more := <-ch:
				if !more {
					continue reconnect
				}
			case <-time.After(pollInterval):
			}
		}
	}
}

func (p *Poller) openWebsocket(ctx context.Context) (chan bool, error) {
	// Do *not* allow panics to bubble up to main. We'll fall back to periodic
	// polling instead.
	defer sentry.RecoverWithContext(ctx)

	if p.wsURL == nil {
		return nil, nil
	}

	log.Printf("discovery: (re-)connecting to websocket...")
	ch := make(chan bool)
	config, err := websocket.NewConfig(p.wsURL.String(), "https://"+p.wsURL.Host)
	if err != nil {
		return nil, err
	}
	if p.cookie.Name != "" {
		// If a cookie is set, send it when opening the Websocket
		config.Header.Add("Cookie", fmt.Sprintf("%s=%s", p.cookie.Name, p.cookie.Value))
	}
	ws, err := websocket.DialConfig(config)
	if err != nil {
		return nil, err
	}
	log.Printf("discovery: opened websocket connection to %q", p.wsURL.String())
	if p.wsToken != "" {
		// Custom (??) authentication protocol from 2021
		data, err := json.Marshal(map[string]interface{}{
			"type": "AUTH",
			"data": p.wsToken,
		})
		if err != nil {
			return nil, err
		}
		if _, err := ws.Write(data); err != nil {
			return nil, err
		}
		log.Printf("discovery: wrote AUTH message to websocket")
	}
	go func(ws *websocket.Conn, ch chan bool) {
		defer close(ch)

		scanner := bufio.NewScanner(ws)
		for scanner.Scan() {
			if p.wsLimiter.Allow() {
				log.Printf("discovery: ws: %q", scanner.Text())
				ch <- true
			} else {
				log.Printf("discovery: ws (skipped due to rate limit): %q", scanner.Text())
			}
		}
		log.Printf("discovery: closing ws channel")
	}(ws, ch)
	return ch, nil
}
