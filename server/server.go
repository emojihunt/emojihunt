package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const sentryContextKey = "emojihunt.sentry"

func Start(ctx context.Context) {
	e := echo.New()
	e.HideBanner = true
	e.Use(SentryMiddleware)
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisablePrintStack: true,
	}))
	e.HTTPErrorHandler = ErrorHandler

	// TODO: robots.txt "User-agent: *\nDisallow: /\n"
	// TODO: reimplement full-resync functionality
	e.GET("/TODO/:id", GetTODO)
	go func() {
		err := e.Start(":8000")
		if !errors.Is(err, http.ErrServerClosed) {
			panic(err)
		}
	}()
	go func() {
		<-ctx.Done()
		e.Shutdown(ctx)
	}()
}

func GetTODO(c echo.Context) error {
	panic("TODO")
}

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

func ErrorHandler(err error, c echo.Context) {
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
			response["sentry_url"] = fmt.Sprintf("TODO: issue URL %s", *event)
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
