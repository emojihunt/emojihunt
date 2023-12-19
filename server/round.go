package server

import (
	"net/http"

	"github.com/emojihunt/emojihunt/db"
	"github.com/labstack/echo/v4"
)

type RoundParams struct {
	ID      int64  `param:"id"`
	Name    string `form:"name"`
	Emoji   string `form:"emoji"`
	Hue     int64  `form:"hue"`
	Special bool   `form:"special"`
}

func (s *Server) ListRounds(c echo.Context) error {
	rounds, err := s.db.ListRounds(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, rounds)
}

func (s *Server) GetRound(c echo.Context) error {
	var id IDParams
	if err := c.Bind(&id); err != nil {
		return err
	}
	round, err := s.db.GetRound(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, round)
}

func (s *Server) CreateRound(c echo.Context) error {
	var params RoundParams
	if err := c.Bind(&params); err != nil {
		return err
	}
	round, err := s.db.CreateRound(c.Request().Context(), db.Round(params))
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, round)
}

func (s *Server) UpdateRound(c echo.Context) error {
	var id IDParams
	if err := c.Bind(&id); err != nil {
		return err
	}
	round, err := s.db.GetRound(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}

	var params = RoundParams(round)
	if err := c.Bind(&params); err != nil {
		return err
	}
	err = s.db.UpdateRound(c.Request().Context(), db.Round(params))
	if err != nil {
		return err
	}

	updated, err := s.db.GetRound(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, updated)
}

func (s *Server) DeleteRound(c echo.Context) error {
	var id IDParams
	if err := c.Bind(&id); err != nil {
		return err
	}
	round, err := s.db.GetRound(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	err = s.db.DeleteRound(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, round)
}
