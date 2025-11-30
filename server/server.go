package server

import (
	"context"
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
	"github.com/emojihunt/emojihunt/util"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	ably    *ably.Realtime
	discord *discord.Client
	echo    *echo.Echo
	state   *state.Client
	sync    *sync.Client

	// authentication and OAuth2 settings
	cookie       *util.SessionCookie
	credentials  *url.Userinfo
	cookieDomain string
	redirectURI  string
}

type IDParams struct {
	ID int64 `param:"id"`
}

func Start(ctx context.Context, prod bool, ably *ably.Realtime,
	discord *discord.Client, state *state.Client, sync *sync.Client) {
	var e = echo.New()
	var s = &Server{
		ably:    ably,
		discord: discord,
		echo:    e,
		state:   state,
		sync:    sync,

		cookie:       util.NewSessionCookie(),
		cookieDomain: util.CookieDomain(prod),
	}

	if raw, ok := os.LookupEnv("OAUTH2_CREDENTIALS"); !ok {
		log.Panicf("OAUTH2_CREDENTIALS is required")
	} else if parts := strings.SplitN(raw, ":", 2); len(parts) != 2 {
		log.Panicf("expected OAUTH2_CREDENTIALS to be CLIENT_ID:CLIENT_SECRET")
	} else {
		s.credentials = url.UserPassword(parts[0], parts[1])
	}

	if prod {
		s.redirectURI = util.ProdAppOrigin + "/login"
	} // else blank redirectURI disables strict validation

	e.HideBanner = true
	e.Use(util.SentryMiddleware)
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		DisablePrintStack: true,
	}))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowCredentials: true,
		AllowOrigins:     []string{util.AppOrigin(prod)},
		ExposeHeaders:    []string{"X-Change-ID"},
	}))
	e.HTTPErrorHandler = util.ErrorHandler(e)

	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"instance": os.Getenv("FLY_MACHINE_VERSION"),
			"status":   "healthy",
			"sync_queues": map[string]interface{}{
				"puzzle": len(s.state.PuzzleChange),
				"round":  len(s.state.RoundChange),
				"live":   len(s.state.LiveMessage),
			},
		})
	})
	e.GET("/robots.txt", func(c echo.Context) error {
		return c.String(http.StatusOK, "User-agent: *\nDisallow: /\n")
	})
	e.POST("/authenticate", s.Authenticate)
	e.POST("/logout", s.Logout)

	var pg = e.Group("/puzzles", s.cookie.AuthenticationMiddleware)
	pg.GET("", s.ListPuzzles)
	pg.GET("/:id", s.GetPuzzle)
	pg.POST("", s.CreatePuzzle)
	pg.POST("/:id", s.UpdatePuzzle)
	pg.DELETE("/:id", s.DeletePuzzle)

	pg.POST("/:id/messages", s.SendMessage)

	var rg = e.Group("/rounds", s.cookie.AuthenticationMiddleware)
	rg.GET("", s.ListRounds)
	rg.GET("/:id", s.GetRound)
	rg.POST("", s.CreateRound)
	rg.POST("/:id", s.UpdateRound)
	rg.DELETE("/:id", s.DeleteRound)

	e.GET("/home", s.ListHome, s.cookie.AuthenticationMiddleware)
	e.POST("/ably", s.RequestAblyToken, s.cookie.AuthenticationMiddleware)
	e.GET("/discovery", s.GetDiscovery, s.cookie.AuthenticationMiddleware)
	e.POST("/discovery", s.UpdateDiscovery, s.cookie.AuthenticationMiddleware)
	e.POST("/discovery/test", s.TestDiscovery, s.cookie.AuthenticationMiddleware)

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

func SetChangeIDHeader(c echo.Context, id int64) {
	c.Response().Header().Set("X-Change-ID", strconv.FormatInt(id, 10))
}
