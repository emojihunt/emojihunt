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
	"github.com/getsentry/sentry-go"
	"golang.org/x/net/websocket"
	"golang.org/x/time/rate"
)

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
	config *state.DiscoveryConfig) *Poller {

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
			if p.state.IsEnabled(ctx) {
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
