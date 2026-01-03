package main

import (
	"log"
	"time"

	"github.com/emojihunt/emojihunt/live/client"
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
		msg, err := client.ReadMessage(ws)
		if err != nil {
			log.Printf("tx: close")
			s.mutex.Lock()
			s.servers -= 1
			s.mutex.Unlock()
			return err
		}
		log.Printf("tx: %#v", msg)
		txMessages.WithLabelValues(string(msg.EventType())).Inc()
		var start = time.Now()
		s.handle(msg)
		handleLatency.Observe(time.Since(start).Seconds())
	}
}

func (s *Server) handle(msg state.LiveMessage) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch v := msg.(type) {
	case client.SettingsMessage:
		s.settings = &v
	case state.AblySyncMessage:
		s.replay = append(s.replay, v)
		if len(s.replay) > 256 {
			s.replay = s.replay[len(s.replay)-256:]
		}
	}

	for _, ch := range s.clients {
		ch <- msg
	}
}
