package main

import (
	"context"
	"log"
	"time"

	"github.com/emojihunt/emojihunt/state"
)

type SheetsMessage struct {
	Sheets  map[string]int64 `json:"sheets"`
	Replace bool             `json:"replace,omitempty"`
}

func (m SheetsMessage) EventType() state.EventType {
	return state.EventTypeSheets
}

func (s *Server) MaybeSendSheetsUpdate() error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	activity, err := s.drive.QueryActivity(ctx)
	if err != nil {
		return err
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	var msg = SheetsMessage{
		Sheets: make(map[string]int64),
	}
	for id, ts := range activity {
		prev, ok := s.sheets[id]
		if !ok || ts.After(prev) {
			msg.Sheets[id] = ts.Unix()
		}
	}
	if len(msg.Sheets) > 0 {
		s.sheets = activity
		log.Printf("sheets: %v", msg.Sheets)

		for _, client := range s.clients {
			client.ch <- msg
		}
	}
	return nil
}
