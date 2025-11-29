package util

import (
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/securecookie"
	"github.com/labstack/echo/v4"
)

const (
	ProdAppOrigin = "https://www.emojihunt.org"
	DevAppOrigin  = "http://localhost:3000"

	ProdCookieDomain = "emojihunt.org"
	DevCookieDomain  = "localhost"

	SessionCookieName = "session"
	SessionDuration   = 4 * 24 * time.Hour
)

func AppOrigin(prod bool) string {
	if prod {
		return ProdAppOrigin
	} else {
		return DevAppOrigin
	}
}

func CookieDomain(prod bool) string {
	if prod {
		return ProdCookieDomain
	} else {
		return DevCookieDomain
	}
}

type SessionCookie struct {
	*securecookie.SecureCookie
}

func NewSessionCookie() *SessionCookie {
	if raw, ok := os.LookupEnv("SERVER_SECRET"); !ok {
		log.Panicf("SERVER_SECRET is required")
	} else if key, err := hex.DecodeString(raw); err != nil || len(key) != 32 {
		log.Panicf("expected SERVER_SECRET to be 32 bytes in hex: %s", err)
	} else {
		var cookie = securecookie.New(key, nil)
		cookie.MaxAge(int(SessionDuration.Seconds()))
		return &SessionCookie{cookie}
	}
	return nil // unreachable
}

func (cookie *SessionCookie) GetUserID(c echo.Context) (string, bool) {
	encoded, err := c.Cookie(SessionCookieName)
	if err != nil {
		return "", false
	}

	var userID string
	err = cookie.Decode(SessionCookieName, encoded.Value, &userID)
	if err != nil {
		log.Printf("invalid session cookie: %v", err)
		return "", false
	}
	return userID, true
}

func (s *SessionCookie) AuthenticationMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, ok := s.GetUserID(c)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid session cookie")
		}
		return next(c)
	}
}
