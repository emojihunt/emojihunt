package db

import (
	"context"
	"database/sql"
	"errors"
	"os"

	_ "embed"

	_ "github.com/mattn/go-sqlite3"
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
