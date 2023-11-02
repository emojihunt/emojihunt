package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"

	_ "net/http/pprof"

	"github.com/emojihunt/emojihunt/bot"
	"github.com/emojihunt/emojihunt/client"
	"github.com/emojihunt/emojihunt/db"
	"github.com/emojihunt/emojihunt/discovery"
	"github.com/emojihunt/emojihunt/server"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
)

type Config struct {
	Discord       *client.DiscordConfig      `json:"discord"`
	GoogleDrive   *client.DriveConfig        `json:"google_drive"`
	Server        *server.ServerConfig       `json:"server"`
	Autodiscovery *discovery.DiscoveryConfig `json:"autodiscovery"`
}

var (
	config_file = flag.String("config", "config.json", "path to the configuration file")
	state_file  = flag.String("state", "state.json", "path to the state file")
	database    = flag.String("db", "db.sqlite", "path to the database file")
)

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
		bs, err = os.ReadFile(*config_file)
		if err != nil {
			log.Fatalf("error reading config file at %q: %v", os.Args[1], err)
		}
	}
	config := Config{}
	if err := json.Unmarshal(bs, &config); err != nil {
		log.Fatalf("error unmarshaling config from %q: %v", os.Args[1], err)
	}

	// Load state
	state, err := state.Load(*state_file)
	if err != nil {
		log.Fatalf("error reading state file at %q: %v", os.Args[2], err)
	}

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
	db := db.OpenDatabase(ctx, *database)

	// Set up clients
	discord, err := client.NewDiscord(config.Discord, state)
	if err != nil {
		log.Fatalf("error creating discord client: %v", err)
	}
	defer discord.Close()

	drive, err := client.NewDrive(ctx, config.GoogleDrive)
	if err != nil {
		log.Fatalf("error creating drive integration: %v", err)
	}

	// Start internal engines
	syncer := syncer.New(db, discord, drive)

	bot.RegisterEmojiNameBot(discord)
	bot.RegisterHuntYetBot(discord)
	bot.RegisterPuzzleBot(ctx, db, discord, syncer)
	bot.RegisterQMBot(discord)
	bot.RegisterReminderBot(db, discord, state)
	bot.RegisterVoiceRoomBot(ctx, db, discord, syncer)

	var dscvpoller *discovery.Poller
	if config.Autodiscovery != nil {
		dscvpoller = discovery.New(db, discord, syncer, config.Autodiscovery, state)
		dscvpoller.RegisterReactionHandler(discord)
	} else {
		log.Printf("puzzle auto-discovery is disabled (no config found)")
	}

	bot.RegisterHuntbotCommand(ctx, db, discord, dscvpoller, syncer, state)

	go func() {
		if err := discord.RegisterCommands(); err != nil {
			log.Fatalf("failed to register discord commands: %v", err)
		}
	}()

	// Run!
	log.Print("press ctrl+C to exit")
	go syncer.RestorePlaceholderEvent()
	if dscvpoller != nil {
		go dscvpoller.Poll(ctx)
	}

	if config.Server != nil {
		server.Start(db, syncer, config.Server)
	} else {
		log.Printf("no server config found, skipping (for development only!)")
	}

	<-ctx.Done()
}

func init() {
	flag.Parse()
}

func init() {
	flag.Parse()
}
