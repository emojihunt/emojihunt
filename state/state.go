package state

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/emojihunt/emojihunt/schema"
)

type State struct {
	AirtableLastWarn   map[string]time.Time `json:"airtable_last_warn"`
	DiscoveryDisabled  bool                 `json:"discovery_disabled"`
	DiscoveryLastWarn  time.Time            `json:"discovery_last_warn"`
	DiscoveryNewRounds map[string]NewRound  `json:"discovery_new_rounds"`
	HuntbotDisabled    bool                 `json:"huntbot_disabled"`
	ReminderTimestamp  time.Time            `json:"reminder_timestamp"`
	ReminderWarnError  time.Time            `json:"reminder_warn_error"`

	mutex    sync.Mutex `json:"-"`
	filename string     `json:"-"`
}

type NewRound struct {
	MessageID string
	Puzzles   []schema.NewPuzzle
}

func Load(filename string) (*State, error) {
	data, err := os.ReadFile(filename)
	if os.IsNotExist(err) {
		err = os.WriteFile(filename, []byte("{}\n"), 0640)
		if err != nil {
			return nil, err
		}
		data, err = os.ReadFile(filename)
	}
	if err != nil {
		return nil, err
	}
	var state State
	err = json.Unmarshal(data, &state)
	if state.AirtableLastWarn == nil {
		state.AirtableLastWarn = make(map[string]time.Time)
	}
	if state.DiscoveryNewRounds == nil {
		state.DiscoveryNewRounds = make(map[string]NewRound)
	}
	state.filename = filename
	return &state, err
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
	err = os.WriteFile(s.filename, data, 0644)
	if err != nil {
		panic(err)
	}
}

func (s *State) IsKilled() bool {
	s.Lock()
	defer s.Unlock()
	return s.DiscoveryDisabled
}
