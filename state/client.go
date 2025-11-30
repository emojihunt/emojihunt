package state

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sync"

	_ "embed"

	"github.com/emojihunt/emojihunt/state/db"
	"golang.org/x/xerrors"
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
		PuzzleChange:    make(chan PuzzleChange, 32),
		RoundChange:     make(chan RoundChange, 8),
		LiveMessage:     make(chan LiveMessage, 32),
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
