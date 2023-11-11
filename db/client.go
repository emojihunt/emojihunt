package db

import (
	"context"
	"database/sql"
	"errors"
	"os"

	_ "embed"

	"github.com/mattn/go-sqlite3"
	"golang.org/x/xerrors"
)

//go:embed schema.sql
var ddl string

type Client struct {
	queries *Queries
}

func OpenDatabase(ctx context.Context, path string) *Client {
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
		if _, err := dbx.ExecContext(ctx, ddl); err != nil {
			panic(xerrors.Errorf("ExecContext(ctx, ddl): %w", err))
		}
	}
	return &Client{New(dbx)}
}

// Unfortunately, sqlite3.Error can't be used with errors.Is/As. This helper
// checks if an error wraps a sqlite3.Error and extracts the (positive) extended
// error code if so. Otherwise, it returns zero.
//
// See: https://github.com/mattn/go-sqlite3/issues/949
func ErrorCode(err error) sqlite3.ErrNoExtended {
	if s, ok := err.(sqlite3.Error); ok {
		return s.ExtendedCode
	} else if e, ok := err.(interface{ Unwrap() []error }); ok {
		for _, err := range e.Unwrap() {
			if c := ErrorCode(err); c > 0 {
				return c
			}
		}
	} else if e, ok := err.(interface{ Unwrap() error }); ok {
		return ErrorCode(e.Unwrap())
	}
	return 0
}
