package main

import (
	"log"
	"net/http"
	"time"

	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/live/client"
	"github.com/emojihunt/emojihunt/state"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/sync/errgroup"
)

var (
	txMessages = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "live_messages",
		Help: "The total number of messages received",
	}, []string{"event"})
	handleLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "live_handle_time",
		Help: "The duration of message handling, in seconds",
	})
)

func (s *Server) Transmit(c echo.Context) error {
	// We only allow one server to connect at a time, and we clear the rewind
	// buffer when it disconnects. This prevents gaps or backwards jumps in the
	// rewind buffer.
	//
	// Note that clients remain connected when the server leaves. We need separate
	// logic to preserve ordering there.
	//
	log.Printf("tx: connect")
	s.mutex.Lock()
	if s.server {
		log.Printf("tx: server already connected")
		s.mutex.Unlock() // unlock in both branches!
		return echo.NewHTTPError(http.StatusConflict, "server already connected")
	} else {
		s.server = true
		s.mutex.Unlock() // unlock in both branches!
	}

	defer func() {
		s.mutex.Lock()
		s.server = false
		s.rewind = nil
		for _, client := range s.clients {
			client.ch <- nil // sentinel indicating a potential discontinuity
		}
		s.mutex.Unlock()
		log.Printf("tx: close")
	}()

	ws, err := s.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	erg, ctx := errgroup.WithContext(c.Request().Context())
	ws.SetReadDeadline(time.Now().Add(client.PongWait))
	ws.SetPongHandler(func(appData string) error {
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
	erg.Go(func() error {
		for {
			msg, err := client.ReadMessage(ws)
			if err != nil {
				return err
			}
			s.handle(msg)
		}
	})
	return erg.Wait()
}

func (s *Server) handle(msg state.LiveMessage) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var start = time.Now()
	defer func() {
		handleLatency.Observe(time.Since(start).Seconds())
	}()
	txMessages.WithLabelValues(string(msg.EventType())).Inc()

	switch v := msg.(type) {
	case *client.SettingsMessage:
		log.Printf("tx: %#v", msg)
		s.settings = v
	case *state.AblySyncMessage:
		if v.Puzzle != nil {
			log.Printf("tx: puzzle: %d %s %#v", v.ChangeID, v.Kind, v.Puzzle)
		} else {
			log.Printf("tx: round: %d %s %#v", v.ChangeID, v.Kind, v.Round)
		}
		if len(s.rewind) > 0 && v.ChangeID <= s.rewind[len(s.rewind)-1].ChangeID {
			log.Printf("tx: out-of-order from %d", s.rewind[len(s.rewind)-1].ChangeID)
		} else {
			s.rewind = append(s.rewind, v)
			if len(s.rewind) > 256 {
				s.rewind = s.rewind[len(s.rewind)-256:]
			}
		}
	case *discord.UsersMessage:
		log.Printf("tx: %#v", v)
		if v.Replace {
			for k := range s.users {
				delete(s.users, k)
			}
		}
		for uid, user := range v.Users {
			s.users[uid] = user
		}
		for _, uid := range v.Delete {
			delete(s.users, uid)
		}
	default:
		log.Printf("tx: unknown: %#v", v)
	}

	for _, client := range s.clients {
		client.ch <- msg
	}
}
