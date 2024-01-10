package discovery

import (
	"bufio"
	"bytes"
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
	"github.com/emojihunt/emojihunt/state"
	"github.com/getsentry/sentry-go"
	"golang.org/x/net/html"
	"golang.org/x/net/websocket"
	"golang.org/x/time/rate"
	"golang.org/x/xerrors"
)

type Poller struct {
	puzzlesURL *url.URL
	cookie     *http.Cookie

	groupMode          bool
	groupSelector      cascadia.Selector
	roundNameSelector  cascadia.Selector
	puzzleListSelector cascadia.Selector
	puzzleItemSelector cascadia.Selector

	wsURL     *url.URL
	wsToken   string
	wsLimiter *rate.Limiter
}

const (
	pollInterval       = 20 * time.Second
	pollTimeout        = 90 * time.Second
	roundCreationPause = 10 * time.Second
	websocketBurst     = 3
)

var websocketRate = rate.Every(1 * time.Minute)

func NewPoller(config state.DiscoveryConfig) (*Poller, error) {
	puzzlesURL, err := url.Parse(config.PuzzlesURL)
	if err != nil {
		return nil, state.ValidationError{Field: "puzzles_url", Message: err.Error()}
	}
	var wsURL *url.URL
	if config.WebsocketURL != "" {
		wsURL, err = url.Parse(config.WebsocketURL)
		if err != nil {
			return nil, state.ValidationError{Field: "websocket_url", Message: err.Error()}
		}
	}

	groupSelector, err := cascadia.Compile(config.GroupSelector)
	if err != nil {
		return nil, state.ValidationError{Field: "group_selector", Message: err.Error()}
	}
	roundNameSelector, err := cascadia.Compile(config.RoundNameSelector)
	if err != nil {
		return nil, state.ValidationError{Field: "round_name_selector", Message: err.Error()}
	}
	puzzleListSelector, err := cascadia.Compile(config.PuzzleListSelector)
	if err != nil {
		return nil, state.ValidationError{Field: "puzzle_name_selector", Message: err.Error()}
	}
	itemSelector := config.PuzzleItemSelector
	if itemSelector == "" {
		itemSelector = "a"
	}
	puzzleItemSelector, err := cascadia.Compile(config.PuzzleItemSelector)
	if err != nil {
		return nil, state.ValidationError{Field: "puzzle_item_selector", Message: err.Error()}
	}

	return &Poller{
		puzzlesURL: puzzlesURL,
		cookie: &http.Cookie{
			Name:   config.CookieName,
			Value:  config.CookieValue,
			MaxAge: 0,
		},

		groupMode:          config.GroupMode,
		groupSelector:      groupSelector,
		roundNameSelector:  roundNameSelector,
		puzzleListSelector: puzzleListSelector,
		puzzleItemSelector: puzzleItemSelector,

		wsURL:     wsURL,
		wsToken:   config.WebsocketToken,
		wsLimiter: rate.NewLimiter(websocketRate, websocketBurst),
	}, nil
}

func (p *Poller) Poll(ctx context.Context, r chan []state.ScrapedPuzzle) error {
	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", "discovery.poll")
	})
	ctx = sentry.SetHubOnContext(ctx, hub)

reconnect:
	for {
		ws, ch, err := p.OpenWebsocket(ctx)
		if err != nil {
			log.Printf("discovery: failed to open websocket: %v", spew.Sprint(err))
		} else if ws != nil {
			defer ws.Close()
		}

		for {
			subctx, cancel := context.WithTimeout(ctx, pollTimeout)
			if puzzles, err := p.Scrape(subctx); err != nil {
				sentry.GetHubFromContext(ctx).CaptureException(err)
			} else {
				r <- puzzles
			}
			cancel()

			select {
			case <-ctx.Done():
				return nil
			case _, more := <-ch:
				if !more {
					continue reconnect
				}
			case <-time.After(pollInterval):
			}
		}
	}
}

func (p *Poller) Scrape(ctx context.Context) ([]state.ScrapedPuzzle, error) {
	// Download
	log.Printf("discovery: scraping %q", p.puzzlesURL.String())
	req, err := http.NewRequestWithContext(ctx, "GET", p.puzzlesURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.AddCookie(p.cookie)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	} else if res.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("failed to fetch puzzle list: status code %v", res.Status)
	}

	// Parse round structure
	var discovered [][2]*html.Node
	root, err := html.Parse(res.Body)
	if err != nil {
		return nil, err
	}

	if p.groupMode {
		groups := p.groupSelector.MatchAll(root)
		if len(groups) == 0 {
			return nil, xerrors.Errorf("no groups found")
		}

		for _, group := range groups {
			nameNode := p.roundNameSelector.MatchFirst(group)
			if nameNode == nil {
				return nil, xerrors.Errorf("round name node not found in group: %#v", group)
			}

			puzzleListNode := p.puzzleListSelector.MatchFirst(group)
			if puzzleListNode == nil {
				return nil, xerrors.Errorf("puzzle list node not found in group: %#v", group)
			}
			discovered = append(discovered, [2]*html.Node{nameNode, puzzleListNode})
		}
	} else {
		container := p.groupSelector.MatchFirst(root)
		if container == nil {
			return nil, xerrors.Errorf("container not found, did login succeed?")
		}

		node := container.FirstChild
		for {
			if p.roundNameSelector.Match(node) {
				nameNode := node

				node = node.NextSibling
				for node.Type == html.TextNode {
					// Skip over text nodes.
					node = node.NextSibling
				}
				if p.puzzleListSelector.Match(node) {
					// Puzzle list found!
					discovered = append(discovered, [2]*html.Node{nameNode, node})
				} else if p.roundNameSelector.Match(node) {
					// Another round heading! This is probably a sub-round;
					// start over treating the new heading as the round name.
					continue
				} else {
					// Unknown structure, abort.
					return nil, xerrors.Errorf("puzzle table not found, got: %#v", node)
				}
			}

			// Advance to next node.
			node = node.NextSibling
			if node == nil {
				break
			}
		}

		if len(discovered) == 0 {
			return nil, xerrors.Errorf("no rounds found in container: %#v", container)
		}
	}

	// Parse out individual puzzles
	var puzzles []state.ScrapedPuzzle
	for _, pair := range discovered {
		nameNode, puzzleListNode := pair[0], pair[1]
		var roundBuf bytes.Buffer
		collectText(nameNode, &roundBuf)
		roundName := strings.TrimSpace(roundBuf.String())

		puzzleItemNodes := p.puzzleItemSelector.MatchAll(puzzleListNode)
		if len(puzzleItemNodes) == 0 {
			return nil, xerrors.Errorf("no puzzle item nodes found in puzzle list: %#v", puzzleListNode)
		}
		for _, item := range puzzleItemNodes {
			var puzzleBuf bytes.Buffer
			collectText(item, &puzzleBuf)

			var u *url.URL
			for _, attr := range item.Attr {
				if attr.Key == "href" {
					u, err = url.Parse(attr.Val)
					if err != nil {
						return nil, xerrors.Errorf("invalid puzzle url: %#v", u)
					}
				}
			}
			if u == nil {
				return nil, xerrors.Errorf("could not find puzzle url for puzzle: %#v", item)
			}

			url := p.puzzlesURL.ResolveReference(u).String()
			puzzles = append(puzzles, state.ScrapedPuzzle{
				Name:      strings.TrimSpace(puzzleBuf.String()),
				RoundName: roundName,
				PuzzleURL: url,
			})
		}
	}
	return puzzles, nil
}

func (p *Poller) OpenWebsocket(ctx context.Context) (*websocket.Conn, chan bool, error) {
	// Do *not* allow panics to bubble up to main. We'll fall back to periodic
	// polling instead.
	defer sentry.RecoverWithContext(ctx)

	if p.wsURL == nil {
		return nil, nil, nil
	}

	log.Printf("discovery: (re-)connecting to websocket...")
	ch := make(chan bool)
	config, err := websocket.NewConfig(p.wsURL.String(), "https://"+p.wsURL.Host)
	if err != nil {
		return nil, nil, err
	}
	if p.cookie.Name != "" {
		// If a cookie is set, send it when opening the Websocket
		config.Header.Add("Cookie", fmt.Sprintf("%s=%s", p.cookie.Name, p.cookie.Value))
	}
	ws, err := websocket.DialConfig(config)
	if err != nil {
		return nil, nil, err
	}
	log.Printf("discovery: opened websocket connection to %q", p.wsURL.String())
	if p.wsToken != "" {
		// Custom (??) authentication protocol from 2021
		data, err := json.Marshal(map[string]interface{}{
			"type": "AUTH",
			"data": p.wsToken,
		})
		if err != nil {
			return nil, nil, err
		}
		if _, err := ws.Write(data); err != nil {
			return nil, nil, err
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
	return ws, ch, nil
}

func collectText(n *html.Node, buf *bytes.Buffer) {
	// https://stackoverflow.com/a/18275336
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, buf)
	}
}
