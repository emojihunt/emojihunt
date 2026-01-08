package main

import (
	"github.com/emojihunt/emojihunt/state"
)

type PresenceMessage map[int64]map[string]bool // puzzle -> usr -> active

func (m PresenceMessage) EventType() state.EventType {
	return state.EventTypePresence
}

func (s *Server) MaybeSendPresenceUpdate() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.presenceChanged {
		return
	}

	var msg = make(PresenceMessage)
	for _, client := range s.clients {
		for puzzle, active := range client.presence {
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
	s.presenceChanged = false
}
