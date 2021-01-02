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
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/discord"
	"github.com/gauravjsingh/emojihunt/drive"
	"github.com/gauravjsingh/emojihunt/huntbot"
	"github.com/gauravjsingh/emojihunt/huntbot/handler"
)

var (
	secretsFile = flag.String("secrets_file", "secrets.json", "path to the flie that contains secrets used by the application")
	sheetID     = flag.String("sheet_id", "1SgvhTBeVdyTMrCR0wZixO3O0lErh4vqX0--nBpSfYT8", "the id of the puzzle tracking sheet to use")
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

// TODO: Move to huntbot.
func registerHandlers(dg *discordgo.Session) {
	// Only handle new guild messages.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages)
	for _, h := range handler.DiscordHandlers {
		dg.AddHandler(h)
	}
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

	err = dg.Open()
	defer dg.Close()
	if err != nil {
		log.Fatalf("error opening discord connection: %v", err)
	}

	// TODO: Move to huntbot.
	registerHandlers(dg)

	dis, err := discord.New(dg, discord.Config{QMChannelName: "bot-testing", ArchiveChannelName: "archive"})
	if err != nil {
		log.Fatalf("error creating discord client: %v", err)
	}
	if err := dis.ArchiveChannel("to-be-archived"); err != nil {
		log.Fatalf("error archiving channel: %v", err)
	}

	ctx := context.Background()

	d, err := drive.New(ctx, secrets.GoogleDriveAPIKey, *sheetID)
	if err != nil {
		log.Fatalf("error creating test drive integration: %v", err)
	}
	huntbot.New(dis, d)

	log.Print("bot is running, press ctrl+C to exit")
	// TODO: use a context instead, pass that along.
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}

func init() {
	flag.Parse()
}
