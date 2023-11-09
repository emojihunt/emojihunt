package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "net/http/pprof"

	"github.com/emojihunt/emojihunt/bot"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/discord"
	"github.com/emojihunt/emojihunt/discovery"
	"github.com/emojihunt/emojihunt/drive"
	"github.com/emojihunt/emojihunt/server"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
	"github.com/getsentry/sentry-go"
)

type Config struct {
	Production    bool                       `json:"production"`
	Sentry        *sentry.ClientOptions      `json:"sentry"`
	Discord       *discord.Config            `json:"discord"`
	GoogleDrive   *drive.Config              `json:"google_drive"`
	Autodiscovery *discovery.DiscoveryConfig `json:"autodiscovery"`
}

var (
	configPath = flag.String("config", "config.json", "path to the configuration file")
	dbPath     = flag.String("db", "db.sqlite", "path to the database file")
)

func init() { flag.Parse() }

func main() {
	// Load configuration
	var bs []byte
	if raw, ok := os.LookupEnv("HUNTBOT_CONFIG"); ok {
		// In production, configuration is stored in a secret (environment
		// variable).
		bs = []byte(raw)
	} else {
		// In development, configuration is stored in a local file.
		var err error
		if bs, err = os.ReadFile(*configPath); err != nil {
			panic(err)
		}
	}
	config := Config{}
	if err := json.Unmarshal(bs, &config); err != nil {
		panic(err)
	}

	// Initialize Sentry
	config.Sentry.BeforeSend = func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
		if hint.OriginalException != nil {
			log.Printf("error: %s", hint.OriginalException)
		} else {
			log.Printf("error: %s", hint.RecoveredException)
		}
		for _, exception := range event.Exception {
			frames := exception.Stacktrace.Frames
			for i := len(frames) - 1; i >= 0; i-- {
				log.Printf("\t%s:%d", frames[i].AbsPath, frames[i].Lineno)
			}
		}
		return event
	}
	if config.Production {
		config.Sentry.Environment = "prod"
	} else {
		config.Sentry.Environment = "dev"
	}

	if err := sentry.Init(*config.Sentry); err != nil {
		panic(err)
	}
	defer sentry.Flush(time.Second * 5)
	defer func() {
		if err := recover(); err != nil {
			sentry.CurrentHub().Recover(err)
			panic(err)
		}
	}()
	// TODO: set up context, error handling in all goroutines

	// Set up the main context, which is cancelled on Ctrl-C
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	go func() { <-ch; cancel() }()

	// Start debug server
	// http://localhost:6060/debug/pprof/goroutine?debug=2
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	// Open database connection
	db := db.OpenDatabase(ctx, *dbPath)

	// Load state
	state, err := state.Load(ctx, db)
	if err != nil {
		panic(err)
	}

	// Set up clients
	discord, err := discord.Connect(ctx, config.Discord, state)
	if err != nil {
		panic(err)
	}
	defer discord.Close()

	drive, err := drive.NewClient(ctx, config.GoogleDrive)
	if err != nil {
		panic(err)
	}

	// Start internal engines
	log.Printf("starting syncer")
	syncer := syncer.New(db, discord, drive)
	go syncer.RestorePlaceholderEvent()

	var dscvpoller *discovery.Poller
	if config.Autodiscovery != nil {
		log.Printf("starting puzzle auto-discovery poller")
		dscvpoller = discovery.New(ctx, db, discord, syncer, config.Autodiscovery, state)
		dscvpoller.RegisterReactionHandler(discord)
		go dscvpoller.Poll(ctx)
	} else {
		log.Printf("puzzle auto-discovery is disabled (no config found)")
	}

	log.Printf("starting web server")
	server.Start(ctx)

	log.Printf("starting discord bots")
	discord.RegisterBots(
		bot.NewEmojiNameBot(),
		bot.NewHuntYetBot(),
		bot.NewHuntBot(ctx, db, discord, dscvpoller, syncer, state),
		bot.NewPuzzleBot(db, discord, syncer),
		bot.NewQMBot(discord),
		bot.NewReminderBot(ctx, db, discord, state),
		bot.NewVoiceRoomBot(ctx, db, discord, syncer),
	)

	log.Print("press ctrl+C to exit")
	<-ctx.Done()
}
