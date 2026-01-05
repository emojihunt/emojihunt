package main

import (
	"github.com/emojihunt/emojihunt/state"
)

type ActivityMessage map[int64]map[string]bool

func (m ActivityMessage) EventType() state.EventType {
	return state.EventTypeActivity
}

func (s *Server) SendActivityUpdate() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var msg = make(ActivityMessage)
	for _, client := range s.clients {
		for puzzle, active := range client.activity {
			if _, ok := msg[puzzle]; !ok {
				msg[puzzle] = make(map[string]bool)
			}
			if !msg[puzzle][client.user] {
				msg[puzzle][client.user] = active
			}
		}
	}

	for _, client := range s.clients {
		client.ch <- msg
	}
}
