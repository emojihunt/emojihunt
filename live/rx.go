package main

import (
	"log"

	"github.com/labstack/echo/v4"
)

func (s *Server) Receive(c echo.Context) error {
	ws, err := s.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	s.mutex.Lock()
	if s.settings != nil {
		ws.WriteJSON(s.settings)
	}
	s.counter += 1
	var id = s.counter
	s.clients[id] = ws
	s.mutex.Unlock()

	log.Printf("rx: connect: %d", id)
	defer func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		log.Printf("rx: close: %d", id)
		delete(s.clients, id)
	}()

	for {
		// Per the docs, we need to read messages in order for ping/pong/close
		// handling to work.
		_, _, err = ws.ReadMessage()
		if err != nil {
			return err
		}
	}
}
