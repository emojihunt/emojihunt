package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"time"

	"github.com/emojihunt/emojihunt/util"
	"github.com/getsentry/sentry-go"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/xerrors"
)

var prod = flag.Bool("prod", false, "selects development or production")

func init() { flag.Parse() }

func main() {
	// Initialize Sentry
	util.SentryInit()
	defer sentry.Flush(time.Second * 5)
	defer func() {
		if err := recover(); err != nil {
			sentry.CurrentHub().Recover(err)
			panic(err)
		}
	}()

	// Debug Server: http://localhost:7070/debug/pprof/goroutine?debug=2
	go func() {
		http.ListenAndServe("localhost:7070", nil)
	}()

	// Set up the main context, which is cancelled on Ctrl-C
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() { <-ch; cancel() }()

	// Start web server
	var e = echo.New()
	var cookie = util.NewSessionCookie()

	e.HideBanner = true
	e.Use(util.SentryMiddleware)
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisablePrintStack: true,
	}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowCredentials: true,
		AllowOrigins:     []string{util.AppOrigin(*prod)},
		ExposeHeaders:    []string{"X-Change-ID"},
	}))
	e.HTTPErrorHandler = util.ErrorHandler(e)

	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"instance": os.Getenv("FLY_MACHINE_VERSION"),
			"status":   "healthy",
		})
	})
	e.GET("/robots.txt", func(c echo.Context) error {
		return c.String(http.StatusOK, "User-agent: *\nDisallow: /\n")
	})
	e.GET("/todo", func(c echo.Context) error {
		user, ok := cookie.GetUserID(c)
		if !ok {
			return xerrors.Errorf("unauthenticated user")
		}
		return c.JSON(http.StatusOK, map[string]interface{}{
			"hello": user,
		})
	}, cookie.AuthenticationMiddleware)

	go func() {
		err := e.Start(":9090")
		if !errors.Is(err, http.ErrServerClosed) {
			log.Panicf("echo.Start: %s", err)
		}
	}()
	go func() {
		<-ctx.Done()
		e.Shutdown(ctx)
	}()

	log.Print("press ctrl+C to exit")
	<-ctx.Done()
}
