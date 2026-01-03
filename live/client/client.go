package client

import (
	"context"
	"errors"
	"log"
	"net/http"
	"syscall"
	"time"

	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/util"
	"github.com/getsentry/sentry-go"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"
)

const (
	ProdLiveURL = "ws://huntlive.internal:9090/tx"
	DevLiveURL  = "ws://localhost:9090/tx"
)

func LiveURL(prod bool) string {
	if prod {
		return ProdLiveURL
	} else {
		return DevLiveURL
	}
}

type Client struct {
	url     string
	dialer  *websocket.Dialer
	discord *discord.Client
	token   string
	state   *state.Client
}

func New(prod bool, discord *discord.Client, state *state.Client) *Client {
	return &Client{
		url: LiveURL(prod),
		dialer: &websocket.Dialer{
			HandshakeTimeout: 30 * time.Second,
		},
		discord: discord,
		token:   util.HuntbotToken(),
		state:   state,
	}
}

func (c *Client) Watch(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	hub := sentry.CurrentHub().Clone()
	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag("task", "live")
	})
	ctx = sentry.SetHubOnContext(ctx, hub)
	// *do* allow panics to bubble up to main()

reconnect:
	for {
		log.Printf("live: connecting...")
		err := c.watch(ctx)
		if errors.Is(err, syscall.ECONNREFUSED) {
			// don't report to Sentry, it will eat our rate limit
		} else if _, ok := err.(*websocket.CloseError); ok {
			log.Printf("live: disconnected")
			// don't report ordinary disconnections
		} else {
			log.Printf("live: %#v", err)
			sentry.GetHubFromContext(ctx).CaptureException(err)
		}

		var wait = time.After(5 * time.Second)
		for {
			select {
			case <-c.state.LiveMessage:
				// drain pending messages, we'll do a full re-sync when we reconnect
				continue
			case <-wait:
				continue reconnect
			case <-ctx.Done():
				return
			}
		}
	}
}

func (c *Client) watch(ctx context.Context) error {
	erg, ctx := errgroup.WithContext(ctx)

	var headers = make(http.Header)
	headers.Add(echo.HeaderAuthorization, c.token)
	ws, _, err := c.dialer.DialContext(ctx, c.url, headers)
	if err != nil {
		return err
	}
	log.Printf("live: connected!")
	defer ws.Close()

	// Per the docs, we need to read messages in order for ping/pong/close
	// handling to work.
	erg.Go(func() error {
		for {
			_, _, err := ws.NextReader()
			if err != nil {
				return err
			}
		}
	})

	// Forward current discovery state
	config, err := c.state.DiscoveryConfig(ctx)
	if err != nil {
		return err
	}
	err = WriteMessage(ws, c.ComputeMeta(config))
	if err != nil {
		return err
	}

	// Forward all past changes in on-disk buffer
	var latest int64
	changes, err := c.state.Changes(ctx)
	if err != nil {
		return err
	}
	for _, change := range changes {
		if change.ChangeID <= latest {
			log.Printf("got out-of-order sync from db")
			continue
		}
		latest = change.ChangeID
		err := WriteMessage(ws, change)
		if err != nil {
			return err
		}
	}

	// Forward incremental updates
	erg.Go(func() error {
		for {
			select {
			case msg := <-c.state.LiveMessage:
				if s, ok := msg.(state.AblySyncMessage); ok {
					if s.ChangeID <= latest {
						log.Printf("out-of-order sync: %#v@%d", s, latest)
						continue
					}
					latest = s.ChangeID
				}
				err := WriteMessage(ws, msg)
				if err != nil {
					return err
				}
			case <-ctx.Done():
				return nil
			}
		}
	})
	return erg.Wait()
}
