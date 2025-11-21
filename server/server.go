package server

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/ably/ably-go/ably"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/sync"
	"github.com/getsentry/sentry-go"
	"github.com/gorilla/securecookie"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mattn/go-sqlite3"
)

type Server struct {
	ably    *ably.Realtime
	discord *discord.Client
	echo    *echo.Echo
	state   *state.Client
	sync    *sync.Client

	// authentication and OAuth2 settings
	cookie       *securecookie.SecureCookie
	credentials  *url.Userinfo
	cookieDomain string
	redirectURI  string
}

type IDParams struct {
	ID int64 `param:"id"`
}

const sentryContextKey = "emojihunt.sentry"

func Start(ctx context.Context, prod bool, ably *ably.Realtime,
	discord *discord.Client, state *state.Client, sync *sync.Client) {
	var e = echo.New()
	var s = &Server{ably: ably, discord: discord, echo: e, state: state, sync: sync}

	if raw, ok := os.LookupEnv("SERVER_SECRET"); !ok {
		log.Panicf("SERVER_SECRET is required")
	} else if key, err := hex.DecodeString(raw); err != nil || len(key) != 32 {
		log.Panicf("expected SERVER_SECRET to be 32 bytes in hex: %s", err)
	} else {
		s.cookie = securecookie.New(key, nil)
		s.cookie.MaxAge(int(SessionDuration.Seconds()))
	}

	if raw, ok := os.LookupEnv("OAUTH2_CREDENTIALS"); !ok {
		log.Panicf("OAUTH2_CREDENTIALS is required")
	} else if parts := strings.SplitN(raw, ":", 2); len(parts) != 2 {
		log.Panicf("expected OAUTH2_CREDENTIALS to be CLIENT_ID:CLIENT_SECRET")
	} else {
		s.credentials = url.UserPassword(parts[0], parts[1])
	}

	var appOrigin string
	if prod {
		appOrigin = ProdAppOrigin
		s.cookieDomain = ProdCookieDomain
		s.redirectURI = ProdAppOrigin + "/login"
	} else {
		appOrigin = DevAppOrigin
		s.cookieDomain = DevCookieDomain
		// blank redirectURI disables strict validation
	}

	e.HideBanner = true
	e.Use(s.SentryMiddleware)
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisablePrintStack: true,
	}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowCredentials: true,
		AllowOrigins:     []string{appOrigin},
		ExposeHeaders:    []string{"X-Change-ID"},
	}))
	e.HTTPErrorHandler = s.ErrorHandler

	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"instance": os.Getenv("FLY_MACHINE_VERSION"),
			"status":   "healthy",
			"sync_queues": map[string]interface{}{
				"puzzle": len(s.state.PuzzleChange),
				"round":  len(s.state.RoundChange),
			},
		})
	})
	e.GET("/robots.txt", func(c echo.Context) error {
		return c.String(http.StatusOK, "User-agent: *\nDisallow: /\n")
	})
	e.POST("/authenticate", s.Authenticate)
	e.POST("/logout", s.Logout)

	var pg = e.Group("/puzzles", s.AuthenticationMiddleware)
	pg.GET("", s.ListPuzzles)
	pg.GET("/:id", s.GetPuzzle)
	pg.POST("", s.CreatePuzzle)
	pg.POST("/:id", s.UpdatePuzzle)
	pg.DELETE("/:id", s.DeletePuzzle)

	pg.POST("/:id/messages", s.SendMessage)

	var rg = e.Group("/rounds", s.AuthenticationMiddleware)
	rg.GET("", s.ListRounds)
	rg.GET("/:id", s.GetRound)
	rg.POST("", s.CreateRound)
	rg.POST("/:id", s.UpdateRound)
	rg.DELETE("/:id", s.DeleteRound)

	e.GET("/home", s.ListHome, s.AuthenticationMiddleware)
	e.POST("/ably", s.RequestAblyToken, s.AuthenticationMiddleware)
	e.GET("/discovery", s.GetDiscovery, s.AuthenticationMiddleware)
	e.POST("/discovery", s.UpdateDiscovery, s.AuthenticationMiddleware)
	e.POST("/discovery/test", s.TestDiscovery, s.AuthenticationMiddleware)

	go func() {
		err := e.Start(":8080")
		if !errors.Is(err, http.ErrServerClosed) {
			log.Panicf("echo.Start: %s", err)
		}
	}()
	go func() {
		<-ctx.Done()
		e.Shutdown(ctx)
	}()
}

func (s *Server) SentryMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		hub := sentry.CurrentHub().Clone()
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetRequest(c.Request())
			scope.SetTag("task", "server")
			scope.SetTag("method", c.Request().Method)
			scope.SetTag("route", c.Path())
		})
		c.Set(sentryContextKey, hub)
		return next(c)
	}
}

func (s *Server) ErrorHandler(err error, c echo.Context) {
	var ve state.ValidationError
	var se sqlite3.Error
	if _, ok := err.(*echo.HTTPError); ok {
	} else if ok := errors.As(err, &ve); ok {
		err = echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else if ok := errors.As(err, &se); ok && se.Code == sqlite3.ErrConstraint {
		err = echo.NewHTTPError(http.StatusBadRequest, err.Error())
	} else if errors.Is(err, sql.ErrNoRows) {
		err = echo.NewHTTPError(http.StatusNotFound, err.Error())
	} else {
		// Report unexpected errors to Sentry
		hub, ok := c.Get(sentryContextKey).(*sentry.Hub)
		if !ok {
			hub = sentry.CurrentHub().Clone()
		}
		hub.CaptureException(err)
	}
	s.echo.DefaultHTTPErrorHandler(err, c)
}

func SetChangeIDHeader(c echo.Context, id int64) {
	c.Response().Header().Set("X-Change-ID", strconv.FormatInt(id, 10))
}
