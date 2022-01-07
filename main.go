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

var (
	secretsFile = flag.String("secrets_file", "secrets.json", "path to the flie that contains secrets used by the application")
)

type secrets struct {
	Airtable      *client.AirtableConfig     `json:"airtable"`
	Discord       *client.DiscordConfig      `json:"discord"`
	GoogleDrive   *client.DriveConfig        `json:"google_drive"`
	Server        *server.ServerConfig       `json:"server"`
	Autodiscovery *discovery.DiscoveryConfig `json:"autodiscovery"`
}

func loadSecrets(path string) (secrets, error) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return secrets{}, fmt.Errorf("error reading secrets file at %q: %v", path, err)
	}
	s := secrets{}
	if err := json.Unmarshal(bs, &s); err != nil {
		return secrets{}, fmt.Errorf("error unmarshaling secrets from %q: %v", path, err)
	}
	return s, nil
}

func main() {
	// Load secrets.json
	secrets, err := loadSecrets(*secretsFile)
	if err != nil {
		log.Fatalf("error loading secrets: %v", err)
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
	discord, err := client.NewDiscord(secrets.Discord)
	if err != nil {
		log.Fatalf("error creating discord client: %v", err)
	}
	defer discord.Close()

	airtable := client.NewAirtable(secrets.Airtable)

	drive, err := client.NewDrive(ctx, secrets.GoogleDrive)
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
		bot.MakeSolveCommand(ctx, airtable, discord, syncer),
		bot.MakeStatusCommand(ctx, airtable, discord, syncer),
		bot.MakeVoiceRoomCommand(airtable, discord),
	}
	var dscvpoller *discovery.Poller
	if secrets.Autodiscovery != nil {
		dscvpoller = discovery.New(airtable, discord, syncer, secrets.Autodiscovery)
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
	go dscvpoller.Poll(ctx)

	if secrets.Server != nil {
		server.Start(airtable, syncer, secrets.Server)
	} else {
		log.Printf("no server config found, skipping (for development only!)")
	}

	<-ctx.Done()
}

func init() {
	flag.Parse()
}
