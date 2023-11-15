package server

import (
	"net/http"

	"github.com/emojihunt/emojihunt/db"
	"github.com/labstack/echo/v4"
)

func (s *Server) ListRounds(c echo.Context) error {
	rounds, err := s.db.ListRounds(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, rounds)
}

func (s *Server) GetRound(c echo.Context) error {
	id, err := parseID(c.Param("id"))
	if err != nil {
		return err
	}
	round, err := s.db.GetRound(c.Request().Context(), id)
	if err != nil {
		return translateError(err)
	}
	return c.JSON(http.StatusOK, round)
}

type CreateRoundParams struct {
	Name  string `form:"name"`
	Emoji string `form:"emoji"`
}

func (s *Server) CreateRound(c echo.Context) error {
	var params CreateRoundParams
	if err := c.Bind(&params); err != nil {
		return err
	}
	round, err := s.db.CreateRound(c.Request().Context(), db.Round{
		Name:  params.Name,
		Emoji: params.Emoji,
	})
	if err != nil {
		return translateError(err)
	}
	return c.JSON(http.StatusOK, round)
}

func (s *Server) UpdateRound(c echo.Context) error {
	id, err := parseID(c.Param("id"))
	if err != nil {
		return err
	}
	round, err := s.db.GetRound(c.Request().Context(), id)
	if err != nil {
		return translateError(err)
	}
	if err = parseParams(c, &round); err != nil {
		return err
	}
	err = s.db.UpdateRound(c.Request().Context(), round)
	if err != nil {
		return translateError(err)
	}
	return c.JSON(http.StatusOK, round)
}

func (s *Server) DeleteRound(c echo.Context) error {
	id, err := parseID(c.Param("id"))
	if err != nil {
		return err
	}
	round, err := s.db.GetRound(c.Request().Context(), id)
	if err != nil {
		return translateError(err)
	}
	err = s.db.DeleteRound(c.Request().Context(), id)
	if err != nil {
		return translateError(err)
	}
	return c.JSON(http.StatusOK, round)
}
