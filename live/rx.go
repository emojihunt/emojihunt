package main

import (
	"log"

	"github.com/emojihunt/emojihunt/state"
	"github.com/labstack/echo/v4"
	"golang.org/x/sync/errgroup"
)

func (s *Server) Receive(c echo.Context) error {
	ws, err := s.upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	defer ws.Close()

	s.mutex.Lock()
	var ch = make(chan *state.LiveMessage, 257)
	if s.settings != nil {
		ch <- s.settings
	}
	for _, msg := range s.sync {
		ch <- msg
	}
	s.counter += 1
	var id = s.counter
	s.clients[id] = ch
	s.mutex.Unlock()

	log.Printf("rx[%04d]: connect", id)
	defer func() {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		log.Printf("rx[%04d]: close", id)
		delete(s.clients, id)
	}()
	defer ws.Close()

	var latest int64
	erg, ctx := errgroup.WithContext(c.Request().Context())
	// Per the docs, we need to read messages in order for ping/pong/close
	// handling to work.
	erg.Go(func() error {
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				return err
			}
		}
	})
	erg.Go(func() error {
		for {
			select {
			case msg := <-ch:
				if msg.ChangeID > 0 {
					if msg.ChangeID <= latest {
						log.Printf("rx[%04d]: out of order: %#v@%d",
							id, msg, latest)
						continue
					}
					latest = msg.ChangeID
				}
				err := ws.WriteJSON(msg)
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
