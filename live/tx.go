package main

import (
	"log"
	"time"

	"github.com/emojihunt/emojihunt/state"
	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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
	log.Printf("tx: connect")
	s.mutex.Lock()
	s.servers += 1
	s.mutex.Unlock()

	for {
		var msg state.LiveMessage
		err = ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("tx: close")
			s.mutex.Lock()
			s.servers -= 1
			s.mutex.Unlock()
			return err
		}
		log.Printf("tx: %#v", msg)
		txMessages.WithLabelValues(string(msg.Event)).Inc()
		var start = time.Now()
		s.handle(&msg)
		handleLatency.Observe(time.Since(start).Seconds())
	}
}

func (s *Server) handle(msg *state.LiveMessage) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch msg.Event {
	case state.EventTypeSettings:
		s.settings = msg
	case state.EventTypeSync:
		s.sync = append(s.sync, msg)
		if len(s.sync) > 256 {
			s.sync = s.sync[len(s.sync)-256:]
		}
	}

	for _, ch := range s.clients {
		ch <- msg
	}
}
