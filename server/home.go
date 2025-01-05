package server

import (
	"net/http"
	"time"

	"github.com/emojihunt/emojihunt/huntyet"
	"github.com/labstack/echo/v4"
)

func (s *Server) ListHome(c echo.Context) error {
	puzzles, rounds, changeID, discovery, err := s.state.ListHome(c.Request().Context())
	if err != nil {
		return err
	}
	next, _ := huntyet.NextHunt(time.Now())
	voiceRooms := s.discord.ListVoiceChannels()
	return c.JSON(http.StatusOK, map[string]interface{}{
		"puzzles":          puzzles,
		"rounds":           rounds,
		"change_id":        changeID,
		"discord_guild":    s.discord.Guild.ID,
		"hanging_out":      s.discord.HangingOutChannel.ID,
		"hunt_name":        discovery.HuntName,
		"hunt_url":         discovery.HuntURL,
		"hunt_credentials": discovery.HuntCredentials,
		"logistics_url":    discovery.LogisticsURL,
		"next_hunt":        next,
		"voice_rooms":      voiceRooms,
	})
}
