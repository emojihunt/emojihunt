package main

import (
	"log"
	"maps"
	"net/http"
	"strconv"
	"time"

	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/live/client"
	"github.com/emojihunt/emojihunt/state"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"
)

type SyncMessage struct {
	Event    string         `json:"event"`
	Activity map[int64]bool `json:"activity"`
}

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

	// Look up snowflake from cookie
	user, _ := s.cookie.GetUserID(c)

	// Access global state (no blocking or returns!)
	s.mutex.Lock()

	// ...set up channel and queue initial messages...
	var ch = make(chan state.LiveMessage, 384)
	if s.settings != nil {
		ch <- *s.settings
	}
	var found = false
	var hasRewind = len(s.rewind) > 0
	for _, msg := range s.rewind {
		if msg.ChangeID == after {
			found = true
		} else if msg.ChangeID > after { // always true if after == 0
			ch <- msg
		}
	}

	var users = make(map[string][2]string)
	maps.Copy(users, s.users)
	ch <- &discord.UsersMessage{
		Users:   users,
		Replace: true,
	}

	var sheets = make(map[string]int64)
	for k, v := range s.sheets {
		sheets[k] = v.Unix()
	}
	ch <- &SheetsMessage{
		Sheets:  sheets,
		Replace: true,
	}

	// ...add ourselves to the global client list...
	s.cctr += 1
	var id = s.cctr
	log.Printf("rx[%04d]: connect %s", id, user)
	s.clients[id] = &Client{ch, make(map[int64]bool), user}
	defer func() {
		s.mutex.Lock()
		log.Printf("rx[%04d]: close", id)
		if len(s.clients[id].presence) > 0 {
			s.presenceChanged = true
		}
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
		return ws.WriteControl(websocket.CloseMessage, msg, time.Now().Add(client.WriteWait))
	}

	// By default, Go sends TCP keepalives every 15 seconds and closes the socket
	// after 9 (?) are missed. However, those keepalives probably go to the Fly
	// proxy, which has its own idle behavior: it supposedly closes connections
	// after 60 seconds of inactivity, although the closures we've observed have
	// actually been 90-300 seconds later, or often not at all.
	erg, ctx := errgroup.WithContext(c.Request().Context())
	ping := ws.PingHandler()
	ws.SetPingHandler(func(appData string) error {
		log.Printf("[rx%04d]: ping %x", id, appData) // TODO: remove me!
		return ping(appData)
	})
	ws.SetReadDeadline(time.Now().Add(client.PongWait))
	ws.SetPongHandler(func(appData string) error {
		log.Printf("rx[%04d]: pong", id) // TODO: remove me!
		ws.SetReadDeadline(time.Now().Add(client.PongWait))
		return nil
	})
	erg.Go(func() error {
		for {
			select {
			case <-time.After(client.PingPeriod):
				err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(client.WriteWait))
				if err != nil {
					return err
				}
			case <-ctx.Done():
				return nil
			}
		}
	})

	// Per the docs, we need to read messages in order for ping/pong/close
	// handling to work.
	erg.Go(func() error {
		for {
			var msg SyncMessage
			err := ws.ReadJSON(&msg)
			if err != nil {
				return err
			}

			log.Printf("rx[%04d]: %v", id, msg)
			switch msg.Event {
			case "activity":
				s.mutex.Lock()
				s.clients[id].presence = msg.Activity
				s.presenceChanged = true
				s.mutex.Unlock()
				activityPings.Inc()
			default:
				log.Printf("rx[%04d]: unknown event type: %s", id, msg.Event)
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
				switch msg.EventType() {
				case state.EventTypeSync:
					var v = msg.(*state.AblySyncMessage)
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
							log.Printf("rx[%04d]: out of order: %v@%d", id, msg, latest)
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
