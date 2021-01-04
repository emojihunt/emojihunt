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
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/discord"
	"github.com/gauravjsingh/emojihunt/drive"
	"github.com/gauravjsingh/emojihunt/huntbot"
	"github.com/gauravjsingh/emojihunt/huntbot/emojiname"
	"github.com/gauravjsingh/emojihunt/huntbot/huntyet"
)

var (
	secretsFile = flag.String("secrets_file", "secrets.json", "path to the flie that contains secrets used by the application")
	sheetID     = flag.String("sheet_id", "1SgvhTBeVdyTMrCR0wZixO3O0lErh4vqX0--nBpSfYT8", "the id of the puzzle tracking sheet to use")
)

type secrets struct {
	DiscordToken string `json:"discord_token"`
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

	err = dg.Open()
	defer dg.Close()
	if err != nil {
		log.Fatalf("error opening discord connection: %v", err)
	}

	dis, err := discord.New(dg, discord.Config{QMChannelName: "bot-testing", SolvedCategoryName: "solved", PuzzleCategoryName: "puzzles"})
	if err != nil {
		log.Fatalf("error creating discord client: %v", err)
	}
	if err := dis.ArchiveChannel("to-be-archived"); err != nil {
		log.Fatalf("error archiving channel: %v", err)
	}

	ctx := context.Background()

	ctx, cancel := context.WithCancel(ctx)
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

	d, err := drive.New(ctx, *sheetID)
	if err != nil {
		log.Fatalf("error creating test drive integration: %v", err)
	}
	h := huntbot.New(dis, d)

	log.Print("press ctrl+C to exit")
	dis.RegisterNewMessageHandler("emoji generator", emojiname.Handler)
	dis.RegisterNewMessageHandler("isithuntyet?", huntyet.Handler)
	dis.RegisterNewMessageHandler("new puzzle", h.NewPuzzleHandler)

	// we don't have a way to subscribe to updates, so we just poll the sheet
	for {
		select {
		case <-ctx.Done():
			log.Print("exiting due to signal")
			return
		case <-time.After(1 * time.Second):
			h.PollSheet(ctx)
		}
	}
}

func init() {
	flag.Parse()
}
