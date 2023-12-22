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
	return c.JSON(http.StatusOK, puzzle)
}

func (s *Server) CreatePuzzle(c echo.Context) error {
	var params PuzzleParams
	if err := c.Bind(&params); err != nil {
		return err
	}
	puzzle, err := s.state.CreatePuzzle(c.Request().Context(), state.RawPuzzle(params))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, puzzle)
}

func (s *Server) UpdatePuzzle(c echo.Context) error {
	var id IDParams
	if err := c.Bind(&id); err != nil {
		return err
	}

	updated, err := s.state.UpdatePuzzle(c.Request().Context(), id.ID,
		func(puzzle *state.RawPuzzle) error {
			var params = (*PuzzleParams)(puzzle)
			return c.Bind(params)
		},
	)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, updated)
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
	err = s.state.DeletePuzzle(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, puzzle)
}
