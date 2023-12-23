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
	DiscoveryChange chan bool
	PuzzleChange    chan PuzzleChange
	RoundChange     chan RoundChange

	queries *db.Queries
	mutex   sync.Mutex // used to serialize writes
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
	return &Client{
		DiscoveryChange: make(chan bool, 8),
		PuzzleChange:    make(chan PuzzleChange, 32),
		RoundChange:     make(chan RoundChange, 8),
		queries:         db.New(dbx),
	}
}

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s %s", e.Field, e.Message)
}