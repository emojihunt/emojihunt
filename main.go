package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"

	_ "net/http/pprof"

	"github.com/emojihunt/emojihunt/bot"
	"github.com/emojihunt/emojihunt/client"
	"github.com/emojihunt/emojihunt/database"
	"github.com/emojihunt/emojihunt/discovery"
	"github.com/emojihunt/emojihunt/server"
	"github.com/emojihunt/emojihunt/state"
	"github.com/emojihunt/emojihunt/syncer"
)

type Config struct {
	Airtable      *client.AirtableConfig     `json:"airtable"`
	Discord       *client.DiscordConfig      `json:"discord"`
	GoogleDrive   *client.DriveConfig        `json:"google_drive"`
	Server        *server.ServerConfig       `json:"server"`
	Autodiscovery *discovery.DiscoveryConfig `json:"autodiscovery"`
}

func main() {
	if len(os.Args) != 3 {
		fmt.Printf("usage: %s CONFIG_FILE STATE_FILE\n", os.Args[0])
		os.Exit(2)
	}

	// Load config.json
	bs, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("error reading secrets file at %q: %v", os.Args[1], err)
	}
	config := Config{}
	if err := json.Unmarshal(bs, &config); err != nil {
		log.Fatalf("error unmarshaling secrets from %q: %v", os.Args[1], err)
	}

	// Load state.json
	state, err := state.Load(os.Args[2])
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

	// Set up clients
	discord, err := client.NewDiscord(config.Discord, state)
	if err != nil {
		log.Fatalf("error creating discord client: %v", err)
	}
	defer discord.Close()

	airtable := client.NewAirtable(config.Airtable)

	drive, err := client.NewDrive(ctx, config.GoogleDrive)
	if err != nil {
		log.Fatalf("error creating drive integration: %v", err)
	}

	// Start internal engines
	syncer := syncer.New(airtable, discord, drive)
	dbpoller := database.NewPoller(airtable, discord, syncer, state)

	bot.RegisterEmojiNameBot(discord)
	bot.RegisterHuntYetBot(discord)
	bot.RegisterPuzzleBot(ctx, airtable, discord, syncer)
	bot.RegisterQMBot(discord)
	bot.RegisterReminderBot(airtable, discord, state)
	bot.RegisterVoiceRoomBot(ctx, airtable, discord, syncer)

	var dscvpoller *discovery.Poller
	if config.Autodiscovery != nil {
		dscvpoller = discovery.New(airtable, discord, syncer, config.Autodiscovery, state)
		dscvpoller.RegisterApproveCommand(ctx, discord)
	} else {
		log.Printf("puzzle auto-discovery is disabled (no config found)")
	}

	bot.RegisterHuntbotCommand(ctx, airtable, discord, dbpoller, dscvpoller, syncer, state)

	if err := discord.RegisterCommands(); err != nil {
		log.Fatalf("failed to register discord commands: %v", err)
	}

	// Run!
	log.Print("press ctrl+C to exit")
	go dbpoller.Poll(ctx)
	if dscvpoller != nil {
		go dscvpoller.Poll(ctx)
	}

	if config.Server != nil {
		server.Start(airtable, syncer, config.Server)
	} else {
		log.Printf("no server config found, skipping (for development only!)")
	}

	<-ctx.Done()
}

func init() {
	flag.Parse()
}
