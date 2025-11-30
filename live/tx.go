package main

import (
	"log"

	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

func (s *Server) Transmit(c echo.Context) error {
	ws, err := s.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	for {
		// Write
		err := ws.WriteMessage(websocket.TextMessage, []byte("Hello, huntbot!"))
		if err != nil {
			log.Printf("tx: werr: %v", err)
		}

		// Read
		_, msg, err := ws.ReadMessage()
		if err != nil {
			log.Printf("tx: rerr: %v", err)
		}
		log.Printf("tx: %s\n", msg)
	}
}
