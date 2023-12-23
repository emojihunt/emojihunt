package server

import (
	"net/http"

	"github.com/emojihunt/emojihunt/state"
	"github.com/labstack/echo/v4"
)

type RoundParams struct {
	ID              int64  `param:"id"`
	Name            string `form:"name"`
	Emoji           string `form:"emoji"`
	Hue             int64  `form:"hue"`
	Sort            int64  `form:"sort"`
	Special         bool   `form:"special"`
	DriveFolder     string `form:"drive_folder"`
	DiscordCategory string `form:"discord_category"`
}

func (s *Server) ListRounds(c echo.Context) error {
	rounds, err := s.state.ListRounds(c.Request().Context())
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
	round, err := s.state.GetRound(c.Request().Context(), id.ID)
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
	round, err := s.state.CreateRound(c.Request().Context(), state.Round(params))
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
	updated, err := s.state.UpdateRound(c.Request().Context(), id.ID,
		func(round *state.Round) error {
			var params = (*RoundParams)(round)
			return c.Bind(&params)
		},
	)
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
	round, err := s.state.GetRound(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	err = s.state.DeleteRound(c.Request().Context(), id.ID)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, round)
}
