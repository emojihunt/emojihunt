package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"

	"github.com/gauravjsingh/emojihunt/bot"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/database"
	"github.com/gauravjsingh/emojihunt/discovery"
	"github.com/gauravjsingh/emojihunt/server"
	"github.com/gauravjsingh/emojihunt/syncer"
)

type Config struct {
	Airtable      *client.AirtableConfig     `json:"airtable"`
	Discord       *client.DiscordConfig      `json:"discord"`
	GoogleDrive   *client.DriveConfig        `json:"google_drive"`
	Server        *server.ServerConfig       `json:"server"`
	Autodiscovery *discovery.DiscoveryConfig `json:"autodiscovery"`
}

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("usage: %s CONFIG_FILE\n", os.Args[0])
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

	// Set up clients
	discord, err := client.NewDiscord(config.Discord)
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
	dbpoller := database.NewPoller(airtable, discord, syncer)
	botCommands := []*client.DiscordCommand{
		bot.MakeEmojiNameCommand(),
		bot.MakeHuntYetCommand(),
		bot.MakeQMCommand(discord),
		bot.MakePuzzleCommand(ctx, airtable, discord, syncer),
		bot.MakeVoiceRoomCommand(airtable, discord),
	}
	var dscvpoller *discovery.Poller
	if config.Autodiscovery != nil {
		dscvpoller = discovery.New(airtable, discord, syncer, config.Autodiscovery)
		botCommands = append(botCommands, dscvpoller.MakeApproveCommand(ctx))
	} else {
		log.Printf("puzzle auto-discovery is disabled (no config found)")
	}
	botCommands = append(botCommands,
		bot.MakeDatabaseCommand(discord, dbpoller, dscvpoller),
	)

	if err := discord.RegisterCommands(botCommands); err != nil {
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
