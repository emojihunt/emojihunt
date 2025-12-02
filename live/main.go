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
	"sync"
	"time"

	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/util"
	"github.com/getsentry/sentry-go"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var prod = flag.Bool("prod", false, "selects development or production")

func init() { flag.Parse() }

type Server struct {
	echo     *echo.Echo
	cookie   *util.SessionCookie
	token    string
	upgrader *websocket.Upgrader

	mutex    sync.Mutex // hold while accessing everything below
	counter  int64
	clients  map[int64]*websocket.Conn
	servers  int
	settings *state.LiveMessage // cache the last settings message
}

var (
	rxConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "live_rx_clients",
		Help: "The total number of active Rx connections",
	})
	txConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "live_tx_clients",
		Help: "The total number of active Tx connections",
	})
)

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

	// Debug Server
	// - http://localhost:7070/debug/pprof/goroutine?debug=2
	// - http://localhost:607070670700/metrics
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":7070", nil)

	// Set up the main context, which is cancelled on Ctrl-C
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() { <-ch; cancel() }()

	// Start web server
	var appOrigins = util.AppOrigins(*prod)
	var s = &Server{
		echo:   echo.New(),
		cookie: util.NewSessionCookie(),
		token:  util.HuntbotToken(),
		upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				if r.Header.Get("Origin") == "" {
					return true
				}
				for _, origin := range appOrigins {
					if strings.EqualFold(r.Header.Get("Origin"), origin) {
						return true
					}
				}
				return false
			},
		},
		clients: make(map[int64]*websocket.Conn),
	}
	go func() {
		for {
			s.mutex.Lock()
			rxConnections.Set(float64(len(s.clients)))
			txConnections.Set(float64(s.servers))
			s.mutex.Unlock()
			time.Sleep(5 * time.Second)
		}
	}()

	s.echo.HideBanner = true
	s.echo.Use(util.SentryMiddleware)
	s.echo.Use(echoprometheus.NewMiddleware("echo"))
	s.echo.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisablePrintStack: true,
	}))
	s.echo.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowCredentials: true,
		AllowOrigins:     appOrigins,
	}))
	s.echo.HTTPErrorHandler = s.ErrorHandler

	s.echo.GET("/", func(c echo.Context) error {
		s.mutex.Lock()
		defer s.mutex.Unlock()
		return c.JSON(http.StatusOK, map[string]interface{}{
			"instance": os.Getenv("FLY_MACHINE_VERSION"),
			"status":   "healthy",
			"websocket": map[string]interface{}{
				"rx": len(s.clients),
				"tx": s.servers,
			},
		})
	})
	s.echo.GET("/robots.txt", func(c echo.Context) error {
		return c.String(http.StatusOK, "User-agent: *\nDisallow: /\n")
	})
	s.echo.GET("/tx", s.Transmit, s.HuntbotMiddleware)
	s.echo.GET("/rx", s.Receive, s.cookie.AuthenticationMiddleware)

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

func (s *Server) ErrorHandler(err error, c echo.Context) {
	if _, ok := err.(*websocket.CloseError); ok {
		// Ignore ordinary websocket closure
	} else if _, ok := err.(*echo.HTTPError); ok {
		// Handle 404s and similar errors
		s.echo.DefaultHTTPErrorHandler(err, c)
	} else {
		// Report unexpected errors to Sentry
		hub, ok := c.Get(util.SentryContextKey).(*sentry.Hub)
		if !ok {
			hub = sentry.CurrentHub().Clone()
		}
		hub.CaptureException(err)

		// Don't invoke DefaultHTTPErrorHandler, since we've probably already upgraded
		// the connection.
	}
}
