package server

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type MessageParams struct {
	PuzzleID int64  `param:"id"`
	Message  string `form:"msg"`
}

func (s *Server) SendMessage(c echo.Context) error {
	var params MessageParams
	if err := c.Bind(&params); err != nil {
		return err
	}
	puzzle, err := s.state.GetPuzzle(c.Request().Context(), params.PuzzleID)
	if err != nil {
		return err
	}
	user, _ := s.GetUserID(c)
	s.discord.RelayMessage(puzzle.DiscordChannel, user, params.Message)
	return c.JSON(http.StatusOK, "")
}
