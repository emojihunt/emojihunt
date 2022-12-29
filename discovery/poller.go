package discovery

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andybalholm/cascadia"
	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/client"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
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

	airtable  *client.Airtable
	discord   *client.Discord
	syncer    *syncer.Syncer
	state     *state.State
	wsLimiter *rate.Limiter
}

type DiscoveredPuzzle struct {
	Name  string
	URL   *url.URL
	Round string
}

const (
	pollInterval       = 20 * time.Second
	pollTimeout        = 90 * time.Second
	warnErrorFrequency = 10 * time.Minute
	preCreationPause   = 10 * time.Second
	newPuzzleLimit     = 15
	websocketBurst     = 3
)

var websocketRate = rate.Every(1 * time.Minute)

func New(airtable *client.Airtable, discord *client.Discord, syncer *syncer.Syncer, config *DiscoveryConfig, state *state.State) *Poller {
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

		airtable:  airtable,
		discord:   discord,
		syncer:    syncer,
		state:     state,
		wsLimiter: rate.NewLimiter(websocketRate, websocketBurst),
	}
}

func (d *Poller) Poll(ctx context.Context) {
reconnect:
	for {
		ch, err := d.openWebsocket()
		if err != nil {
			log.Printf("discovery: failed to open websocket: %v", spew.Sprint(err))
		}

		for {
			if !d.isEnabled() {
				time.Sleep(2 * time.Second)
				continue
			}

			d.poll(ctx)

			select {
			case <-ctx.Done():
				log.Print("exiting discovery poller due to signal")
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

func (d *Poller) poll(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, pollTimeout)
	defer cancel()

	puzzles, err := d.Scrape(ctx)
	if err != nil {
		d.logAndMaybeWarn("scraping error", err)
	}

	if err := d.SyncPuzzles(ctx, puzzles); err != nil {
		d.logAndMaybeWarn("syncing error", err)
	}
}

func (d *Poller) isEnabled() bool {
	d.state.Lock()
	defer d.state.Unlock()
	return !d.state.DiscoveryDisabled
}

func (d *Poller) logAndMaybeWarn(memo string, err error) {
	d.state.Lock()
	defer d.state.CommitAndUnlock()

	log.Printf("discovery: %s: %v", memo, err)
	if time.Since(d.state.DiscoveryLastWarn) >= warnErrorFrequency {
		msg := fmt.Sprintf("```*** PUZZLE DISCOVERY %s ***\n\n%s```", strings.ToUpper(memo), spew.Sdump(err))
		d.discord.ChannelSend(d.discord.TechChannel, msg)
		d.state.DiscoveryLastWarn = time.Now()
	}
}

func (d *Poller) openWebsocket() (chan bool, error) {
	if d.wsURL == nil {
		return nil, nil
	}

	log.Printf("discovery: (re-)connecting to websocket...")
	ch := make(chan bool)
	ws, err := websocket.Dial(d.wsURL.String(), "", "https://"+d.wsURL.Host)
	if err != nil {
		return nil, err
	}
	log.Printf("discovery: opened websocket connection to %q", d.wsURL.String())
	data, err := json.Marshal(map[string]interface{}{
		"type": "AUTH",
		"data": d.wsToken,
	})
	if err != nil {
		return nil, err
	}
	log.Printf("discovery: wrote AUTH message to websocket")
	if _, err := ws.Write(data); err != nil {
		return nil, err
	}
	go func(ws *websocket.Conn, ch chan bool) {
		defer close(ch)
		scanner := bufio.NewScanner(ws)
		for scanner.Scan() {
			if d.wsLimiter.Allow() {
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
