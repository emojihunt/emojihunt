package server

import (
	"net/http"

	"github.com/emojihunt/emojihunt/state"
	"github.com/labstack/echo/v4"
)

func (s *Server) ListHome(c echo.Context) error {
	puzzles, rounds, changeID, discovery, err := s.state.ListHome(c.Request().Context())
	if err != nil {
		return err
	}
	var rawPuzzles = make([]state.RawPuzzle, len(puzzles))
	for i, puzzle := range puzzles {
		rawPuzzles[i] = puzzle.RawPuzzle()
	}
	voiceRooms := s.discord.ListVoiceChannels()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"change_id":   changeID,
		"puzzles":     rawPuzzles,
		"rounds":      rounds,
		"settings":    s.sync.ComputeMeta(discovery),
		"voice_rooms": voiceRooms,
	})
}
