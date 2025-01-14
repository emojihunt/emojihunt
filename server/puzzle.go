package server

import (
	"net/http"
	"time"

	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/state/status"
	"github.com/labstack/echo/v4"
)

type PuzzleParams struct {
	ID             int64         `param:"id"`
	Name           string        `form:"name"`
	Answer         string        `form:"answer"`
	Round          int64         `form:"round"`
	Status         status.Status `form:"status"`
	Note           string        `form:"note"`
	Location       string        `form:"location"`
	PuzzleURL      string        `form:"puzzle_url"`
	SpreadsheetID  string        `form:"spreadsheet_id"`
	DiscordChannel string        `form:"discord_channel"`
	Meta           bool          `form:"meta"`
	VoiceRoom      string        `form:"voice_room"`
	Reminder       time.Time     `form:"reminder"`
}

func (s *Server) ListPuzzles(c echo.Context) error {
	puzzles, err := s.state.ListPuzzles(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, puzzles)
}

func (s *Server) GetPuzzle(c echo.Context) error {
	var id IDParams
	if err := c.Bind(&id); err != nil {
		return err
	}
	puzzle, err := s.state.GetPuzzle(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, puzzle.RawPuzzle())
}

func (s *Server) CreatePuzzle(c echo.Context) error {
	var err error
	var ctx = c.Request().Context()

	var params PuzzleParams
	if err = c.Bind(&params); err != nil {
		return err
	}
	var raw = state.RawPuzzle(params)
	round, err := s.state.GetRound(ctx, raw.Round)
	if err != nil {
		return err
	}

	// Run validations before handling sheet and channel creation
	if err = s.state.ValidatePuzzle(ctx, raw); err != nil {
		return err
	}
	if raw.SpreadsheetID == "+" {
		raw.SpreadsheetID = ""
		raw.SpreadsheetID, err = s.sync.CreateSpreadsheet(ctx, raw)
		if err != nil {
			return err
		}
	}
	if raw.DiscordChannel == "+" {
		raw.DiscordChannel = ""
		raw.DiscordChannel, err = s.sync.CreateDiscordChannel(ctx, raw, round)
		if err != nil {
			return err
		}
	}

	puzzle, chid, err := s.state.CreatePuzzle(ctx, raw)
	if err != nil {
		return err
	}
	SetChangeIDHeader(c, chid)
	return c.JSON(http.StatusOK, puzzle.RawPuzzle())
}

func (s *Server) UpdatePuzzle(c echo.Context) error {
	var id IDParams
	if err := c.Bind(&id); err != nil {
		return err
	}
	updated, chid, err := s.state.UpdatePuzzle(c.Request().Context(), id.ID,
		func(puzzle *state.RawPuzzle) error {
			var params = (*PuzzleParams)(puzzle)
			return c.Bind(params)
		},
	)
	if err != nil {
		return err
	}
	SetChangeIDHeader(c, chid)
	return c.JSON(http.StatusOK, updated.RawPuzzle())
}

func (s *Server) DeletePuzzle(c echo.Context) error {
	var id IDParams
	if err := c.Bind(&id); err != nil {
		return err
	}
	puzzle, err := s.state.GetPuzzle(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	chid, err := s.state.DeletePuzzle(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	SetChangeIDHeader(c, chid)
	return c.JSON(http.StatusOK, puzzle.RawPuzzle())
}
