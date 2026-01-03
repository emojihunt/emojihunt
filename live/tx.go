package main

import (
	"log"
	"time"

	"github.com/emojihunt/emojihunt/live/client"
	"github.com/emojihunt/emojihunt/state"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/xerrors"
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
	ws, err := s.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()
	err = s.connect()
	if err != nil {
		return err
	}

	for {
		msg, err := client.ReadMessage(ws)
		if err != nil {
			log.Printf("tx: close")
			s.mutex.Lock()
			s.server = false
			s.mutex.Unlock()
			return err
		}
		s.handle(msg)
	}
}

func (s *Server) connect() error {
	log.Printf("tx: connect")

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.server {
		s.mutex.Unlock()
		return xerrors.Errorf("server already connected")
	}
	s.server = true
	s.rewind = nil // clear rewind buffer to prevent gaps
	return nil
}

func (s *Server) handle(msg state.LiveMessage) {
	log.Printf("tx: %#v", msg)
	txMessages.WithLabelValues(string(msg.EventType())).Inc()
	var start = time.Now()
	defer func() {
		handleLatency.Observe(time.Since(start).Seconds())
	}()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch v := msg.(type) {
	case client.SettingsMessage:
		s.settings = &v
	case state.AblySyncMessage:
		if len(s.rewind) > 0 && v.ChangeID <= s.rewind[len(s.rewind)-1].ChangeID {
			log.Printf("tx: out-of-order from %d", s.rewind[len(s.rewind)-1].ChangeID)
		} else {
			s.rewind = append(s.rewind, v)
			if len(s.rewind) > 256 {
				s.rewind = s.rewind[len(s.rewind)-256:]
			}
		}
	}

	for _, ch := range s.clients {
		ch <- msg
	}
}
