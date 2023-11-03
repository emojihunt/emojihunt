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
	Sentry        *sentry.ClientOptions      `json:"sentry"`
	Discord       *discord.Config            `json:"discord"`
	GoogleDrive   *drive.Config              `json:"google_drive"`
	Server        *server.ServerConfig       `json:"server"`
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
		bs, err = os.ReadFile(*configPath)
		if err != nil {
			log.Panicf("error reading config file at %q: %v", *configPath, err)
		}
	}
	config := Config{}
	if err := json.Unmarshal(bs, &config); err != nil {
		log.Panicf("error unmarshaling config from %q: %v", *configPath, err)
	}

	// Initialize Sentry
	config.Sentry.AttachStacktrace = true
	// config.Sentry.Debug = true // TODO
	if err := sentry.Init(*config.Sentry); err != nil {
		log.Panicf("error initializing Sentry: %v", err)
	}
	defer func() {
		if err := recover(); err != nil {
			sentry.CurrentHub().Recover(err)
			sentry.Flush(time.Second * 5)
			panic(err)
		}
	}()

	// Set up our context, which is cancelled on Ctrl-C
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt)
	defer func() {
		signal.Stop(ch)
		cancel()
	}()
	go func() {
		select {
		case <-ch:
			cancel()
		case <-ctx.Done():
		}
	}()

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
		log.Panicf("error loading state: %v", err)
	}

	// Set up clients
	discord, err := discord.NewClient(config.Discord, state)
	if err != nil {
		log.Panicf("error creating discord client: %v", err)
	}
	defer discord.Close()

	drive, err := drive.NewClient(ctx, config.GoogleDrive)
	if err != nil {
		log.Panicf("error creating drive integration: %v", err)
	}

	// Start internal engines
	syncer := syncer.New(db, discord, drive)
	go syncer.RestorePlaceholderEvent()
	log.Printf("started syncer")

	var dscvpoller *discovery.Poller
	if config.Autodiscovery != nil {
		dscvpoller = discovery.New(db, discord, syncer, config.Autodiscovery, state)
		dscvpoller.RegisterReactionHandler(discord)
		go dscvpoller.Poll(ctx)
		log.Printf("started puzzle auto-discovery poller")
	} else {
		log.Printf("puzzle auto-discovery is disabled (no config found)")
	}

	discord.RegisterBots(
		bot.NewEmojiNameBot(),
		bot.NewHuntYetBot(),
		bot.NewHuntBot(db, discord, dscvpoller, syncer, state),
		bot.NewPuzzleBot(db, discord, syncer),
		bot.NewQMBot(discord),
		bot.NewReminderBot(db, discord, state),
		bot.NewVoiceRoomBot(db, discord, syncer),
	)
	log.Printf("started discord bots")

	if config.Server != nil {
		server.Start(db, syncer, config.Server)
		log.Printf("started web server")
	} else {
		log.Printf("no server config found, skipping (for development only!)")
	}

	log.Print("press ctrl+C to exit")
	<-ctx.Done()
}
