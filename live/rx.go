package main

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/emojihunt/emojihunt/live/client"
	"github.com/emojihunt/emojihunt/state"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"
)

func (s *Server) Receive(c echo.Context) error {
	// Parse `after` GET parameter
	var after int64
	var err error
	raw := c.QueryParam("after")
	if raw != "" {
		after, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "param `after`: not a number")
		}
	}

	// Access global state (no blocking or returns!)
	s.mutex.Lock()

	// ...set up channel and queue initial messages...
	var ch = make(chan state.LiveMessage, 257)
	var found = false
	var hasRewind = len(s.rewind) > 0
	if s.settings != nil {
		ch <- *s.settings
	}
	for _, msg := range s.rewind {
		if msg.ChangeID == after {
			found = true
		} else if msg.ChangeID > after { // always true if after == 0
			ch <- msg
		}
	}

	// ...add ourselves to the global client list...
	s.counter += 1
	var id = s.counter
	log.Printf("rx[%04d]: connect", id)
	s.clients[id] = ch
	defer func() {
		s.mutex.Lock()
		log.Printf("rx[%04d]: close", id)
		delete(s.clients, id)
		s.mutex.Unlock()
	}()

	s.mutex.Unlock()

	// Start websocket!
	ws, err := s.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	// We have to return errors over the socket rather than via HTTP status codes
	// because the browser doesn't expose those :(
	if after > 0 && !found {
		var msg []byte
		if hasRewind {
			msg = websocket.FormatCloseMessage(4004, "change not found in rewind buffer")
		} else {
			msg = websocket.FormatCloseMessage(4005, "no tx server connected")
		}
		return ws.WriteControl(websocket.CloseMessage, msg, time.Now().Add(10*time.Second))
	}

	// Per the docs, we need to read messages in order for ping/pong/close
	// handling to work.
	erg, ctx := errgroup.WithContext(c.Request().Context())
	erg.Go(func() error {
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				return err
			}
		}
	})
	erg.Go(func() error {
		var latest int64
		var searching = false
		for {
			select {
			case msg := <-ch:
				if msg == nil { // discontinuity (server disconnected)
					if latest > 0 {
						searching = true
					}
					continue // skip relaying nil message
				}
				if v, ok := msg.(state.AblySyncMessage); ok {
					if searching {
						// Trying to fix a discontinuity
						if v.ChangeID < latest {
							continue
						} else if v.ChangeID == latest {
							searching = false // we're all caught up!
							continue
						} else { // v.ChangeID > latest
							// The server was gone too long before reconnecting (256 or more
							// changes) so there is a permament gap in the record. (This
							// should be rare.) Force the client to reconnect.
							log.Printf("rx[%04d]: could not pick up at %d, got %d",
								id, latest, v.ChangeID)
							ws.Close()
							return nil
						}
					} else {
						// Relaying messages normally
						if v.ChangeID <= latest {
							log.Printf("rx[%04d]: out of order: %#v@%d", id, msg, latest)
							continue
						}
						latest = v.ChangeID // happy path!
					}
				}
				err := client.WriteMessage(ws, msg)
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
