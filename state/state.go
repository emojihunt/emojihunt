package state

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/emojihunt/emojihunt/db"
)

type State struct {
	DiscoveryLastWarn  time.Time           `json:"discovery_last_warn"`
	DiscoveryNewRounds map[string]NewRound `json:"discovery_new_rounds"`
	HuntbotDisabled    bool                `json:"huntbot_disabled"`
	ReminderTimestamp  time.Time           `json:"reminder_timestamp"`
	ReminderWarnError  time.Time           `json:"reminder_warn_error"`

	mutex sync.Mutex `json:"-"`
	db    *db.Client `json:"-"`
}

type NewRound struct {
	MessageID string
	Puzzles   []DiscoveredPuzzle
}

type DiscoveredPuzzle struct {
	Name  string
	Round string
	URL   string
}

func Load(ctx context.Context, db *db.Client) *State {
	data, err := db.LoadState(ctx)
	if err != nil {
		log.Panicf("could not load state: %s", err)
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		log.Panicf("could not unmarshal state: %s", err)
	}
	if state.DiscoveryNewRounds == nil {
		state.DiscoveryNewRounds = make(map[string]NewRound)
	}
	state.db = db
	return &state
}

func (s *State) Lock() {
	s.mutex.Lock()
}

func (s *State) Unlock() {
	s.mutex.Unlock()
}

func (s *State) CommitAndUnlock() {
	defer s.mutex.Unlock()

	data, err := json.MarshalIndent(&s, "", "  ")
	if err != nil {
		panic(err)
	}
	err = s.db.WriteState(context.TODO(), data)
	if err != nil {
		panic(err)
	}
}

func (s *State) IsKilled() bool {
	s.Lock()
	defer s.Unlock()
	return s.HuntbotDisabled
}
