package server

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"net/http"

	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mattn/go-sqlite3"
	"golang.org/x/xerrors"
)

type Config struct {
	SecretKey string `json:"secret_key"`
}

type Server struct {
	db        *db.Client
	discord   *discord.Client
	echo      *echo.Echo
	secretKey [32]byte
}

type IDParams struct {
	ID int64 `param:"id"`
}

const sentryContextKey = "emojihunt.sentry"

func Start(ctx context.Context, db *db.Client, discord *discord.Client,
	issueURL string, config *Config) {

	var e = echo.New()
	var s = &Server{db: db, discord: discord, echo: e}
	key, err := hex.DecodeString(config.SecretKey)
	if err != nil || len(key) != 32 {
		panic(xerrors.Errorf("expected 32-byte key in hex: %w", err))
	}
	copy(s.secretKey[:], key)

	e.HideBanner = true
	e.Use(s.SentryMiddleware)
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisablePrintStack: true,
	}))
	e.Use(s.AuthenticationMiddleware)
	e.HTTPErrorHandler = s.ErrorHandler

	// TODO: robots.txt "User-agent: *\nDisallow: /\n"
	// TODO: reimplement full-resync functionality
	e.POST("/authenticate", s.Authenticate)

	e.GET("/puzzles", s.ListPuzzles)
	e.GET("/puzzles/:id", s.GetPuzzle)
	e.POST("/puzzles", s.CreatePuzzle)
	e.POST("/puzzles/:id", s.UpdatePuzzle)
	e.DELETE("/puzzles/:id", s.DeletePuzzle)

	e.GET("/rounds", s.ListRounds)
	e.GET("/rounds/:id", s.GetRound)
	e.POST("/rounds", s.CreateRound)
	e.POST("/rounds/:id", s.UpdateRound)
	e.DELETE("/rounds/:id", s.DeleteRound)

	go func() {
		err := e.Start(":8000")
		if !errors.Is(err, http.ErrServerClosed) {
			panic(xerrors.Errorf("echo.Start: %w", err))
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
	var ve db.ValidationError
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
