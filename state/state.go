package state

import (
	"encoding/json"
	"os"
	"sync"
)

type State struct {
	HuntbotDisabled   bool `json:"huntbot_disabled"`
	DiscoveryDisabled bool `json:"discovery_disabled"`

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
