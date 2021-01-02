package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/discord/drive"
	"github.com/gauravjsingh/emojihunt/discord/handler"
	"github.com/gauravjsingh/emojihunt/discord/update"
)

var (
	secretsFile = flag.String("secrets_file", "secrets.json", "path to the flie that contains secrets used by the application")
)

type secrets struct {
	DiscordToken      string `json:"discord_token"`
	GoogleDriveAPIKey string `json:"google_drive_api_key"`
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
	secrets, err := loadSecrets(*secretsFile)
	if err != nil {
		log.Fatalf("error loading secrets: %v", err)
	}
	dg, err := discordgo.New(secrets.DiscordToken)
	if err != nil {
		log.Fatalf("error creating discord client: %v", err)
	}

	// Only handle new guild messages.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)
	for _, h := range handler.DiscordHandlers {
		dg.AddHandler(h)
	}

	err = dg.Open()
	if err != nil {
		log.Fatalf("error opening discord connection: %v", err)
	}

	u, err := update.New(dg, "bot-testing")
	if err != nil {
		log.Fatalf("error creating updater: %v", err)
	}
	_ = u

	if err := drive.ConnectToDrive(secrets.GoogleDriveAPIKey); err != nil {
		log.Fatalf("error creating test drive integration: %v", err)
	}

	log.Print("bot is running, press ctrl+C to exit")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
	dg.Close()
}

func init() {
	flag.Parse()
}
