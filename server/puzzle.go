package server

import (
	"database/sql"
	"net/http"

	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/db/field"
	"github.com/labstack/echo/v4"
)

type PuzzleParams struct {
	ID             int64        `param:"id"`
	Name           string       `form:"name"`
	Answer         string       `form:"answer"`
	Round          int64        `form:"round"`
	Status         field.Status `form:"status"`
	Description    string       `form:"description"`
	Location       string       `form:"location"`
	PuzzleURL      string       `form:"puzzle_url"`
	SpreadsheetID  string       `form:"spreadsheet_id"`
	DiscordChannel string       `form:"discord_channel"`
	OriginalURL    string       `form:"original_url"`
	NameOverride   string       `form:"name_override"`
	Archived       bool         `form:"archived"`
	VoiceRoom      string       `form:"voice_room"`
	Reminder       sql.NullTime `form:"reminder"`
}

func (s *Server) ListPuzzles(c echo.Context) error {
	puzzles, err := s.db.ListPuzzles(c.Request().Context())
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
	puzzle, err := s.db.LoadByID(c.Request().Context(), id.ID)
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
	puzzle, err := s.db.CreatePuzzle(c.Request().Context(), db.RawPuzzle(params))
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
	puzzle, err := s.db.GetRawPuzzle(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}

	var params = PuzzleParams(puzzle)
	if err := c.Bind(&params); err != nil {
		return err
	}
	err = s.db.UpdatePuzzle(c.Request().Context(), db.RawPuzzle(params))
	if err != nil {
		return err
	}

	puzzle2, err := s.db.LoadByID(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, puzzle2)
}

func (s *Server) DeletePuzzle(c echo.Context) error {
	var id IDParams
	if err := c.Bind(&id); err != nil {
		return err
	}
	puzzle, err := s.db.LoadByID(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	err = s.db.DeletePuzzle(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, puzzle)
}
