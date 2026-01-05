package main

import (
	"log"
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
	snowflake, _ := s.cookie.GetUserID(c)

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

	var msg = discord.UsersMessage{
		Users:   make(map[string][2]string),
		Replace: true,
	}
	for k, v := range s.users {
		msg.Users[k] = v
	}
	ch <- &msg

	// ...add ourselves to the global client list...
	s.cctr += 1
	var id = s.cctr
	if _, ok := s.userIds[snowflake]; !ok {
		s.uctr += 1
		s.userIds[snowflake] = s.uctr
	}
	var uid = s.userIds[snowflake]
	log.Printf("rx[%04d]: connect %s/%d", id, snowflake, uid)
	s.clients[id] = &Client{ch, make(map[int64]bool), uid}
	defer func() {
		s.mutex.Lock()
		log.Printf("rx[%04d]: close", id)
		if len(s.clients[id].activity) > 0 {
			s.activityChanged = true
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
		return ws.WriteControl(websocket.CloseMessage, msg, time.Now().Add(10*time.Second))
	}

	// Per the docs, we need to read messages in order for ping/pong/close
	// handling to work.
	erg, ctx := errgroup.WithContext(c.Request().Context())
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
				s.clients[id].activity = msg.Activity
				s.activityChanged = true
				s.mutex.Unlock()
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
