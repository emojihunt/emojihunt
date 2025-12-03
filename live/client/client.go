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
	"golang.org/x/xerrors"
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
		if err != nil {
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
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var headers = make(http.Header)
	headers.Add(echo.HeaderAuthorization, c.token)
	ws, _, err := c.dialer.DialContext(ctx, c.url, headers)
	if err != nil {
		if errors.Is(err, syscall.ECONNREFUSED) {
			return nil // don't report to Sentry, it will eat our rate limit
		}
		return xerrors.Errorf("live: %w", err)
	}
	log.Printf("live: connected!")
	defer ws.Close()

	// Per the docs, we need to read messages in order for ping/pong/close
	// handling to work.
	var fin = make(chan error)
	go func() {
		for {
			if _, _, err := ws.NextReader(); err != nil {
				if _, ok := err.(*websocket.CloseError); ok {
					log.Printf("live: disconnected")
					fin <- nil
				} else {
					log.Printf("live: %#v", err)
					fin <- err
				}
				return
			}
		}
	}()

	// Resynchronize state
	config, err := c.state.DiscoveryConfig(ctx)
	if err != nil {
		return err
	}
	var message = c.ComputeMeta(config)
	err = ws.WriteJSON(
		state.LiveMessage{Event: state.EventTypeSettings, Data: message},
	)
	if err != nil {
		return err
	}

	// Forward incremental updates
	go func() {
		for {
			select {
			case msg := <-c.state.LiveMessage:
				err := ws.WriteJSON(msg)
				if err != nil {
					fin <- err
					return
				}
			case <-ctx.Done():
				fin <- nil
				return
			}
		}
	}()
	return <-fin
}
