package main

import (
	"context"
	"crypto/hmac"
	"errors"
	"flag"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/emojihunt/emojihunt/util"
	"github.com/getsentry/sentry-go"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

var prod = flag.Bool("prod", false, "selects development or production")

func init() { flag.Parse() }

type Server struct {
	echo     *echo.Echo
	cookie   *util.SessionCookie
	token    string
	upgrader *websocket.Upgrader
}

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
	var appOrigin = util.AppOrigin(*prod)
	var s = &Server{
		echo:   echo.New(),
		cookie: util.NewSessionCookie(),
		token:  util.HuntbotToken(),
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return strings.EqualFold(r.Header.Get("Origin"), appOrigin) ||
					r.Header.Get("Origin") == ""
			},
		},
	}

	s.echo.HideBanner = true
	s.echo.Use(util.SentryMiddleware)
	s.echo.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisablePrintStack: true,
	}))
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowCredentials: true,
		AllowOrigins:     []string{appOrigin},
	}))
	s.echo.HTTPErrorHandler = util.ErrorHandler(s.echo)

	s.echo.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"instance": os.Getenv("FLY_MACHINE_VERSION"),
			"status":   "healthy",
		})
	})
	s.echo.GET("/robots.txt", func(c echo.Context) error {
		return c.String(http.StatusOK, "User-agent: *\nDisallow: /\n")
	})
	s.echo.GET("/tx", s.Transmit, s.HuntbotMiddleware)

	go func() {
		err := s.echo.Start(":9090")
		if !errors.Is(err, http.ErrServerClosed) {
			log.Panicf("echo.Start: %s", err)
		}
	}()
	go func() {
		<-ctx.Done()
		s.echo.Shutdown(ctx)
	}()

	log.Print("press ctrl+C to exit")
	<-ctx.Done()
}

func (s *Server) HuntbotMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if hmac.Equal(
			[]byte(c.Request().Header.Get(echo.HeaderAuthorization)),
			[]byte(s.token),
		) {
			return next(c)
		} else {
			return echo.NewHTTPError(http.StatusUnauthorized, "missing or invalid bearer token")
		}
	}
}
