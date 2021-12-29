package huntbot

import (
	"sync"
	"time"

	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/schema"
	"github.com/gauravjsingh/emojihunt/syncer"
)

type Config struct {
	// How often to warn in discord about badly formatted puzzles.
	MinWarningFrequency time.Duration
	InitialWarningDelay time.Duration
}

type HuntBot struct {
	airtable *client.Airtable
	discord  *client.Discord
	syncer   *syncer.Syncer
	cfg      Config

	mu              sync.Mutex               // hold while accessing everything below
	enabled         bool                     // global killswitch, toggle with !huntbot kill/!huntbot start
	puzzleStatus    map[string]schema.Status // name -> status (best-effort cache)
	channelToPuzzle map[string]string        // discord channel -> puzzle name (populated by database loop)
	archived        map[string]bool          // name -> channel was archived (best-effort cache)
	// When we last warned about a malformed puzzle.
	lastWarnTime map[string]time.Time
}

func New(airtable *client.Airtable, discord *client.Discord, syncer *syncer.Syncer, c Config) *HuntBot {
	return &HuntBot{
		airtable:     airtable,
		discord:      discord,
		syncer:       syncer,
		enabled:      true,
		puzzleStatus: map[string]schema.Status{},
		archived:     map[string]bool{},
		lastWarnTime: map[string]time.Time{},
		cfg:          c,
	}
}

func (h *HuntBot) isEnabled() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.enabled
}
