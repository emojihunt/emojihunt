package util

import (
	"log"
	"os"

	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
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

const SentryContextKey = "emojihunt.sentry"

func SentryMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		hub := sentry.CurrentHub().Clone()
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetRequest(c.Request())
			scope.SetTag("task", "server")
			scope.SetTag("method", c.Request().Method)
			scope.SetTag("route", c.Path())
		})
		c.Set(SentryContextKey, hub)
		return next(c)
	}
}
