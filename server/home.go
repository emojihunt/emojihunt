package server

import (
	"net/http"
	"time"

	"github.com/emojihunt/emojihunt/huntyet"
	"github.com/labstack/echo/v4"
)

func (s *Server) ListHome(c echo.Context) error {
	puzzles, err := s.state.ListPuzzles(c.Request().Context())
	if err != nil {
		return err
	}
	rounds, err := s.state.ListRounds(c.Request().Context())
	if err != nil {
		return err
	}
	next, _ := huntyet.NextHunt(time.Now())
	voiceRooms := s.discord.ListVoiceChannels()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"puzzles":     puzzles,
		"rounds":      rounds,
		"next_hunt":   next,
		"voice_rooms": voiceRooms,
	})
}
