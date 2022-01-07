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
	certFile    = flag.String("certificate", "/etc/letsencrypt/live/huntbox.emojihunt.tech/fullchain.pem", "the path to the server certificate")
	keyFile     = flag.String("private_key", "/etc/letsencrypt/live/huntbox.emojihunt.tech/privkey.pem", "the path to the server private key")
	origin      = flag.String("origin", "https://huntbox.emojihunt.tech", "origin of the hunt server, for URLs")
)

type secrets struct {
	Airtable    *client.AirtableConfig `json:"airtable"`
	Discord     *client.DiscordConfig  `json:"discord"`
	GoogleDrive *client.DriveConfig    `json:"google_drive"`

	HuntboxToken string `json:"huntbox_token"`
	CookieName   string `json:"hunt_cookie_name"` // to log in to the Hunt website
	CookieValue  string `json:"hunt_cookie_value"`
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
	dscvpoller := discovery.New(secrets.CookieName, secrets.CookieValue, airtable, discord, syncer)

	err = discord.RegisterCommands([]*client.DiscordCommand{
		bot.MakeDatabaseCommand(discord, dbpoller, dscvpoller),
		bot.MakeEmojiNameCommand(),
		bot.MakeHuntYetCommand(),
		bot.MakeQMCommand(discord),
		bot.MakeSolveCommand(ctx, airtable, discord, syncer),
		bot.MakeStatusCommand(ctx, airtable, discord, syncer),
		bot.MakeVoiceRoomCommand(airtable, discord),
		dscvpoller.MakeApproveCommand(ctx),
	})
	if err != nil {
		log.Fatalf("failed to register discord commands: %v", err)
	}

	// Run!
	log.Print("press ctrl+C to exit")
	go dbpoller.Poll(ctx)
	go dscvpoller.Poll(ctx)

	server := server.New(airtable, syncer, secrets.HuntboxToken, *origin)
	server.Start(*certFile, *keyFile)

	<-ctx.Done()
}

func init() {
	flag.Parse()
}
