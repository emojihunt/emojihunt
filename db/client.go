package db

import (
	"context"
	"database/sql"
	"errors"
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
	_, err := os.Stat(path)
	shouldInitialize := errors.Is(err, os.ErrNotExist)

	dbx, err := sql.Open("sqlite3", path)
	if err != nil {
		panic(err)
	}
	if shouldInitialize {
		if _, err := dbx.ExecContext(ctx, ddl); err != nil {
			panic(err)
		}
	}
	return &Client{New(dbx), &sync.Map{}}
}
