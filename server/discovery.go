package server

import (
	"net/http"

	"github.com/emojihunt/emojihunt/state"
	"github.com/labstack/echo/v4"
)

type DiscoveryParams struct {
	PuzzlesURL         string `form:"puzzles_url"`
	CookieName         string `form:"cookie_name"`
	CookieValue        string `form:"cookie_value"`
	GroupMode          bool   `form:"group_mode"`
	GroupSelector      string `form:"group_selector"`
	RoundNameSelector  string `form:"round_name_selector"`
	PuzzleListSelector string `form:"puzzle_list_selector"`
	PuzzleItemSelector string `form:"puzzle_item_selector"`
	WebsocketURL       string `form:"websocket_url"`
	WebsocketToken     string `form:"websocket_token"`
}

func (s *Server) GetDiscovery(c echo.Context) error {
	config, err := s.state.DiscoveryConfig(c.Request().Context())
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, config)
}

func (s *Server) UpdateDiscovery(c echo.Context) error {
	config, err := s.state.UpdateDiscoveryConfig(c.Request().Context(),
		func(config *state.DiscoveryConfig) error {
			var params = (*DiscoveryParams)(config)
			return c.Bind(params)
		},
	)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, config)
}
