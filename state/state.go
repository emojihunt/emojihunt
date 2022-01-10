package state

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

type State struct {
	HuntbotDisabled    bool                 `json:"huntbot_disabled"`
	DiscoveryDisabled  bool                 `json:"discovery_disabled"`
	DiscoveryLastWarn  time.Time            `json:"discovery_last_warn"`
	DiscoveryNewRounds map[string]time.Time `json:"discovery_new_rounds"`
	AirtableLastWarn   map[string]time.Time `json:"airtable_last_warn"`

	mutex    sync.Mutex `json:"-"`
	filename string     `json:"-"`
}

func Load(filename string) (*State, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var state State
	err = json.Unmarshal(data, &state)
	if state.DiscoveryNewRounds == nil {
		state.DiscoveryNewRounds = make(map[string]time.Time)
	}
	if state.AirtableLastWarn == nil {
		state.AirtableLastWarn = make(map[string]time.Time)
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
