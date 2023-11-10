package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (s *Server) ListPuzzles(c echo.Context) error {
	puzzles, err := s.db.ListPuzzles(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, puzzles)
}
