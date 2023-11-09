package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/rivo/uniseg"
)

func (s *Server) ListRounds(c echo.Context) error {
	rounds, err := s.db.ListRounds(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, rounds)
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
	if params.Name == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "round name is required")
	} else if params.Emoji == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "round emoji is required")
	} else if uniseg.GraphemeClusterCount(params.Emoji) != 1 {
		return echo.NewHTTPError(http.StatusBadRequest, "round emoji must be a single grapheme cluster")
	} else if uniseg.StringWidth(params.Emoji) != 2 {
		// *almost* correct, see https://github.com/rivo/uniseg/issues/27
		return echo.NewHTTPError(http.StatusBadRequest, "round emoji must be an emoji")
	}

	round, err := s.db.CreateRound(c.Request().Context(), params.Name, params.Emoji)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, round)
}
