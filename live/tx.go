package main

import (
	"log"

	"github.com/emojihunt/emojihunt/state"
	"github.com/labstack/echo/v4"
)

func (s *Server) Transmit(c echo.Context) error {
	ws, err := s.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()
	log.Printf("tx: connect")

	for {
		var msg state.LiveMessage
		err = ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("tx: close")
			return err
		}
		log.Printf("tx: %#v", msg)
		s.handle(msg)
	}
}

func (s *Server) handle(msg state.LiveMessage) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if msg.Event == state.EventTypeSettings {
		s.settings = &msg
	}

	for _, ws := range s.clients {
		err := ws.WriteJSON(msg)
		if err != nil {
			log.Printf("client: %#v", err)
			ws.Close()
		}
	}
}
