package server

import (
	"net/http"

	"github.com/ably/ably-go/ably"
	"github.com/labstack/echo/v4"
	"golang.org/x/xerrors"
)

const ablyCapability = `{"huntbot":["subscribe"],"discord":["subscribe"]}`

func (s *Server) RequestAblyToken(c echo.Context) error {
	token, err := s.ably.Auth.RequestToken(
		c.Request().Context(),
		&ably.TokenParams{Capability: ablyCapability},
	)
	if err != nil {
		return xerrors.Errorf("RequestToken: %w", err)
	}

	return c.JSON(http.StatusOK, token)
}
