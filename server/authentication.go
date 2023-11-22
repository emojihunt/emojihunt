package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/nacl/secretbox"
	"golang.org/x/xerrors"
)

const (
	SessionExpiry  = 4 * 24 * time.Hour
	OAuth2TokenURL = "https://discord.com/api/v10/oauth2/token"

	DevRedirectURI  = "http://localhost:3000/login"
	ProdRedirectURI = "https://www.emojihunt.tech/login"
)

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
	Code string `form:"code"`
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
	if params.Code == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "code is required")
	}

	token, err := s.oauth2TokenExchange(params.Code)
	if err != nil {
		log.Printf("OAuth2 token exchange failed: %#v", err)
		return echo.NewHTTPError(http.StatusBadRequest, "token exchange failed")
	}

	session, err := s.discord.GetOAuth2Session(c.Request().Context(), token)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err)
	}
	member, err := s.discord.GetGuildMember(&session.User)
	if err != nil {
		// return username for error ui
		return c.JSON(http.StatusUnauthorized, AuthenticateResponse{
			Username: session.User.Username,
		})
	}

	var nonce [24]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	data := Session{
		DiscordUser: session.User.ID,
		Expiry:      time.Now().Add(SessionExpiry).Round(time.Second),
	}
	inner, err := json.Marshal(data)
	if err != nil {
		return err
	}
	sealed := secretbox.Seal(nonce[:], inner, &nonce, &s.secretKey)

	username := session.User.Username
	if member.Nick != "" {
		username = member.Nick
	}
	return c.JSON(http.StatusOK, AuthenticateResponse{
		APIKey:   base64.RawURLEncoding.EncodeToString(sealed),
		Username: username,
	})
}

func (s *Server) oauth2TokenExchange(code string) (string, error) {
	endpoint, err := url.Parse(OAuth2TokenURL)
	if err != nil {
		return "", err
	}
	endpoint.User = s.credentials

	var query = url.Values{}
	query.Add("grant_type", "authorization_code")
	query.Add("code", code)
	query.Add("redirect_uri", s.redirectURI)

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
