package db

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"sync"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var ddl string

type Client struct {
	queries *Queries

	// A map of ID -> puzzle mutex. The puzzle mutex should be held while
	// reading or writing the puzzle, and should be acquired before the voice
	// room mutex (if needed).
	mutexes *sync.Map
}

func OpenDatabase(ctx context.Context, path string) *Client {
	var fresh bool
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		fresh = true
	}
	dbx, err := sql.Open("sqlite3", path)
	if err != nil {
		log.Fatalf("error opening database at %q: %v", path, err)
	}
	if fresh {
		if ddl == "" {
			log.Fatalf("error reading embeded ddl")
		} else if _, err := dbx.ExecContext(ctx, ddl); err != nil {
			log.Fatalf("error initializing database at %q: %v", path, err)
		}
	}
	return &Client{
		queries: New(dbx),
		mutexes: &sync.Map{},
	}
}
