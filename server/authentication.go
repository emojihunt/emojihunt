package server

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/xerrors"
)

const (
	SessionDuration = 4 * 24 * time.Hour
	OAuth2TokenURL  = "https://discord.com/api/v10/oauth2/token"

	ProdCookieDomain = "emojihunt.org"
	DevCookieDomain  = "localhost"

	ProdAppOrigin = "https://www.emojihunt.org"
	DevAppOrigin  = "http://localhost:3000"

	CookieName = "session"
)

func (s *Server) AuthenticationMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		_, ok := s.GetUserID(c)
		if !ok {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid session cookie")
		}
		return next(c)
	}
}

type AuthenticateParams struct {
	Code        string `form:"code"`
	RedirectURI string `form:"redirect_uri"` // used in dev mode only!
}

type AuthenticateResponse struct {
	Username string `json:"username"`
}

func (s *Server) Authenticate(c echo.Context) error {
	var params AuthenticateParams
	if err := c.Bind(&params); err != nil {
		return err
	}
	if params.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is required")
	}

	token, err := s.oauth2TokenExchange(params.Code, params.RedirectURI)
	if err != nil {
		log.Printf("OAuth2 token exchange failed: %#v", err)
		return c.JSON(http.StatusForbidden, AuthenticateResponse{})
	}

	session, err := s.discord.GetOAuth2Session(c.Request().Context(), token)
	if err != nil {
		return err
	}
	member, err := s.discord.GetGuildMember(&session.User)
	if err != nil {
		// return username for error ui
		return c.JSON(http.StatusForbidden, AuthenticateResponse{
			Username: session.User.Username,
		})
	}

	encoded, err := s.cookie.Encode(CookieName, session.User.ID)
	if err != nil {
		return err
	}
	c.SetCookie(&http.Cookie{
		Name:     CookieName,
		Value:    encoded,
		Expires:  time.Now().Add(SessionDuration - 10*time.Minute),
		Path:     "/",
		Domain:   s.cookieDomain,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	username := session.User.Username
	if member.Nick != "" {
		username = member.Nick
	}
	return c.JSON(http.StatusOK, AuthenticateResponse{
		Username: username,
	})
}

func (s *Server) oauth2TokenExchange(code, devRedirectURI string) (string, error) {
	endpoint, err := url.Parse(OAuth2TokenURL)
	if err != nil {
		return "", err
	}
	endpoint.User = s.credentials

	var query = url.Values{}
	query.Add("grant_type", "authorization_code")
	query.Add("code", code)

	if s.redirectURI == "" {
		query.Add("redirect_uri", devRedirectURI)
	} else {
		query.Add("redirect_uri", s.redirectURI)
	}

	resp, err := http.PostForm(endpoint.String(), query)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	} else if raw, ok := data["access_token"]; !ok {
		return "", xerrors.Errorf("malformed token response: %#v", data)
	} else if token, ok := raw.(string); !ok {
		return "", xerrors.Errorf("malformed token response: %#v", data)
	} else {
		return token, nil
	}
}

func (s *Server) GetUserID(c echo.Context) (string, bool) {
	encoded, err := c.Cookie(CookieName)
	if err != nil {
		return "", false
	}

	var userID string
	err = s.cookie.Decode(CookieName, encoded.Value, &userID)
	if err != nil {
		log.Printf("invalid session cookie: %v", err)
		return "", false
	}
	return userID, true
}

func (s *Server) Logout(c echo.Context) error {
	c.SetCookie(&http.Cookie{
		Name:     CookieName,
		Value:    "",
		Expires:  time.Unix(0, 0),
		Path:     "/",
		Domain:   s.cookieDomain,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return c.NoContent(http.StatusOK)
}
