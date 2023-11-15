package server

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strconv"

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

func parseID(val string) (int64, error) {
	id, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, APIError{Type: ErrInvalidValue, Field: "id"}
	}
	return id, nil
}

func parseParams[T any](c echo.Context, obj T) error {
	params, err := c.FormParams()
	if err != nil {
		return err
	}

	var rt = reflect.TypeOf(obj).Elem()
	var tags = make(map[string]int)
	for i := 0; i < rt.NumField(); i++ {
		tags[rt.Field(i).Tag.Get("json")] = i
	}

	var rv = reflect.ValueOf(obj).Elem()
	for pk, pv := range params {
		if i, ok := tags[pk]; ok {
			rv.Field(i).Set(reflect.ValueOf(pv[0]))
		} else {
			return APIError{Type: ErrUnknownKey, Field: pk}
		}
	}
	return nil
}
