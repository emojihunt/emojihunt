package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/emojihunt/emojihunt/db"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/xerrors"
)

const sentryContextKey = "emojihunt.sentry"

type Server struct {
	db        *db.Client
	sentryURL string
}

func Start(ctx context.Context, db *db.Client, issueURL string) {
	var s = &Server{
		db:        db,
		sentryURL: issueURL,
	}

	var e = echo.New()
	e.HideBanner = true
	e.Use(s.SentryMiddleware)
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisablePrintStack: true,
	}))
	e.HTTPErrorHandler = s.ErrorHandler

	// TODO: robots.txt "User-agent: *\nDisallow: /\n"
	// TODO: reimplement full-resync functionality
	e.GET("/puzzles", s.ListPuzzles)
	e.GET("/rounds", s.ListRounds)
	e.GET("/rounds/:id", s.GetRound)
	e.POST("/rounds", s.CreateRound)
	e.POST("/rounds/:id", s.UpdateRound)

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
	var code = http.StatusInternalServerError
	var response = make(map[string]interface{})

	if he, ok := err.(*echo.HTTPError); ok {
		if he.Internal != nil {
			if herr, ok := he.Internal.(*echo.HTTPError); ok {
				he = herr
			}
		}
		code = he.Code
		response["message"] = he.Message
	} else {
		response["message"] = http.StatusText(http.StatusInternalServerError)
		response["error"] = err.Error()

		// Report unexpected errors to Sentry
		hub, ok := c.Get(sentryContextKey).(*sentry.Hub)
		if !ok {
			hub = sentry.CurrentHub().Clone()
		}
		event := hub.CaptureException(err)
		if event != nil {
			response["sentry_url"] = fmt.Sprintf(s.sentryURL, *event)
		}
	}

	// See https://github.com/labstack/echo/blob/master/echo.go
	if c.Response().Committed {
		return
	}
	if c.Request().Method == http.MethodHead {
		err = c.NoContent(code)
	} else {
		err = c.JSON(code, response)
	}
	if err != nil {
		log.Printf("error replying with error: %v", err)
	}
}
