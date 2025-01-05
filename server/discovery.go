package server

import (
	"net/http"

	"github.com/emojihunt/emojihunt/discovery"
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

	HuntName        string `form:"hunt_name"`
	HuntURL         string `form:"hunt_url"`
	HuntCredentials string `form:"hunt_credentials"`
	LogisticsURL    string `form:"logistics_url"`
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
			err := c.Bind(params)
			if err != nil {
				return err
			}
			if config.PuzzlesURL != "" {
				_, err = discovery.NewPoller(*config) // validate config
			}
			return err
		},
	)
	if err != nil {
		return err
	}
	return c.JSON(http.StatusOK, config)
}

func (s *Server) TestDiscovery(c echo.Context) error {
	config, err := s.state.DiscoveryConfig(c.Request().Context())
	if err != nil {
		return err
	}
	var params = (*DiscoveryParams)(&config)
	err = c.Bind(params)
	if err != nil {
		return err
	}
	poller, err := discovery.NewPoller(config) // validate config
	if err != nil {
		return err
	}
	puzzles, err := poller.Scrape(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, puzzles)
}
