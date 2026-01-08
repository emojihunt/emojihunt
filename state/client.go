package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	_ "embed"

	"github.com/emojihunt/emojihunt/state/db"
	"github.com/labstack/gommon/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/xerrors"
)

var (
	changeId = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "change_id",
		Help: "The current change ID",
	})
	puzzlesUnlocked = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "puzzles_unlocked",
		Help: "The total number of puzzles unlocked",
	})
	puzzlesSolved = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "puzzles_solved",
		Help: "The total number of puzzles solved",
	})
	roundsAvailable = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "rounds",
		Help: "The total number of rounds unlocked",
	})
)

type Client struct {
	// Important: to avoid deadlocks, do not send to this channel while holding
	// the lock below.
	DiscoveryChange chan bool
	PuzzleChange    chan PuzzleChange
	RoundChange     chan RoundChange
	LiveMessage     chan LiveMessage

	queries  *db.Queries
	mutex    sync.Mutex // used to serialize database writes
	changeID int64      // must hold mutex when reading/writing
}

func New(ctx context.Context, path string) *Client {
	_, err := os.Stat(path)
	shouldInitialize := errors.Is(err, os.ErrNotExist)

	dbx, err := sql.Open("sqlite3", path+"?_fk=on")
	if err != nil {
		panic(xerrors.Errorf("sql.Open: %w", err))
	}
	if err := dbx.PingContext(ctx); err != nil {
		panic(xerrors.Errorf("PingContext: %w", err))
	}
	if shouldInitialize {
		if _, err := dbx.ExecContext(ctx, db.DDL); err != nil {
			panic(xerrors.Errorf("ExecContext(ctx, ddl): %w", err))
		}
	}
	var client = Client{
		DiscoveryChange: make(chan bool, 8),
		PuzzleChange:    make(chan PuzzleChange, 256),
		RoundChange:     make(chan RoundChange, 256),
		LiveMessage:     make(chan LiveMessage, 256),
		queries:         db.New(dbx),
	}
	epoch, err := client.IncrementSyncEpoch(ctx)
	if err != nil {
		panic(err)
	}
	// Allow for 4 billion writes per restart. Note: Javascript can safely
	// represent numbers up to ~2^53.
	client.changeID = epoch << 32
	return &client
}

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s %s", e.Field, e.Message)
}

func (c *Client) HandleMetrics() {
	for {
		c.mutex.Lock()
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		changeId.Set(float64(c.changeID))

		stats, err := c.queries.CountPuzzles(ctx)
		if err != nil {
			log.Printf("state: CountPuzzles: %#v", err)
		} else {
			puzzlesUnlocked.Set(float64(stats.Total))
			puzzlesSolved.Set(float64(stats.Solved))
		}

		rounds, err := c.queries.CountRounds(ctx)
		if err != nil {
			log.Printf("state: CountRounds: %#v", err)
		} else {
			roundsAvailable.Set(float64(rounds))
		}

		cancel()
		c.mutex.Unlock()
		time.Sleep(30 * time.Second)
	}
}
