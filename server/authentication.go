package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/nacl/secretbox"
)

const SessionExpiry = 4 * 24 * time.Hour

type Session struct {
	DiscordUser string    `json:"u"`
	Expiry      time.Time `json:"e"`
}

func (s *Server) AuthenticationMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		header := c.Request().Header.Get("Authorization")
		if header == "" {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing Authrorization header")
		}
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "only Bearer tokens are supported")
		}
		if !s.verifyToken(token) {
			return echo.NewHTTPError(http.StatusUnauthorized, "malformed or expired token")
		}
		return next(c)
	}
}

func (s *Server) verifyToken(token string) bool {
	bytes, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return false
	}

	var nonce [24]byte
	copy(nonce[:], bytes[:24])
	inner, ok := secretbox.Open(nil, bytes[24:], &nonce, &s.secretKey)
	if !ok {
		return false
	}

	var session Session
	if err := json.Unmarshal(inner, &session); err != nil {
		return false
	}
	return time.Until(session.Expiry) > 0
}

type AuthenticateParams struct {
	AccessToken string `form:"access_token"`
}

type AuthenticateResponse struct {
	APIKey   string `json:"api_key"`
	Username string `json:"username"`
}

func (s *Server) Authenticate(c echo.Context) error {
	var params AuthenticateParams
	if err := c.Bind(&params); err != nil {
		return err
	}
	if params.AccessToken == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "access_token is required")
	}
	member, err := s.discord.CheckOAuth2Token(c.Request().Context(), params.AccessToken)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}

	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	session := Session{
		DiscordUser: member.User.ID,
		Expiry:      time.Now().Add(SessionExpiry).Round(time.Second),
	}
	inner, err := json.Marshal(session)
	if err != nil {
		return err
	}
	sealed := secretbox.Seal(nonce[:], inner, &nonce, &s.secretKey)

	username := member.User.Username
	if member.Nick != "" {
		username = member.Nick
	}
	return c.JSON(http.StatusOK, AuthenticateResponse{
		APIKey:   base64.RawURLEncoding.EncodeToString(sealed),
		Username: username,
	})
}
