package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"net/http"
	_ "net/http/pprof"

	"github.com/ably/ably-go/ably"
	"github.com/emojihunt/emojihunt/bot"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/discovery"
	"github.com/emojihunt/emojihunt/drive"
	live "github.com/emojihunt/emojihunt/live/client"
	"github.com/emojihunt/emojihunt/server"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
	"github.com/emojihunt/emojihunt/util"
	"github.com/getsentry/sentry-go"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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

	// Debug Server
	// - http://localhost:6060/debug/pprof/goroutine?debug=2
	// - http://localhost:6060/metrics
	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":6060", nil)

	// Set up the main context, which is cancelled on Ctrl-C
	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)

	// Open database connection
	var state = state.New(ctx, "db.sqlite")

	// Set up clients
	ablyKey, ok := os.LookupEnv("ABLY_API_KEY")
	if !ok {
		log.Panicf("ABLY_API_KEY is required")
	}
	ably, err := ably.NewRealtime(ably.WithKey(ablyKey))
	if err != nil {
		log.Panicf("ably.NewRealtime: %s", err)
	}
	defer ably.Close()

	var wg sync.WaitGroup
	var drv *drive.Client
	wg.Go(func() {
		drv = drive.NewClient(ctx, *prod)
	})
	var discord = discord.Connect(ctx, *prod, state, ably, &wg)
	wg.Wait()
	defer discord.Close()

	// Start internal engines
	var live = live.New(*prod, discord, state)
	var syncer = syncer.New(ably, discord, drv, live, state)
	var discovery = discovery.New(discord, state, syncer)

	log.Printf("starting discord bots and handlers")
	discord.RegisterBots(
		bot.NewEmojiNameBot(),
		bot.NewHuntYetBot(),
		bot.NewPuzzleBot(discord, state),
		bot.NewQMBot(discord, state),
		bot.NewReminderBot(ctx, discord, state),
	)
	discord.RegisterHandlers(ctx) // yes, there is a small gap here

	go live.Watch(ctx)
	go syncer.Watch(ctx)
	go discovery.SyncWorker(ctx)
	go discovery.Watch(ctx)

	log.Printf("starting web server")
	server.Start(ctx, *prod, ably, discord, live, state, syncer)

	time.Sleep(1 * time.Second)
	log.Print("press ctrl+C to exit")
	<-ctx.Done()
}
