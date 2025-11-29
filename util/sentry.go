package util

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/emojihunt/emojihunt/state"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
	"github.com/mattn/go-sqlite3"
)

func SentryInit() {
	dsn, ok := os.LookupEnv("SENTRY_DSN")
	if !ok {
		panic("SENTRY_DSN is required")
	}
	sentry.Init(sentry.ClientOptions{
		Dsn: dsn,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			if hint.OriginalException != nil {
				log.Printf("error: %s", hint.OriginalException)
			} else {
				log.Printf("error: %s", hint.RecoveredException)
			}
			for _, exception := range event.Exception {
				if tr := exception.Stacktrace; tr != nil {
					for i := len(tr.Frames) - 1; i >= 0; i-- {
						log.Printf("\t%s:%d", tr.Frames[i].AbsPath, tr.Frames[i].Lineno)
					}
				}
			}
			return event
		},
	})
}

const sentryContextKey = "emojihunt.sentry"

func SentryMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
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

func ErrorHandler(e *echo.Echo) func(error, echo.Context) {
	return func(err error, c echo.Context) {
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
		e.DefaultHTTPErrorHandler(err, c)
	}
}
