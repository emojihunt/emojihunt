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
	secretsFile  = flag.String("secrets_file", "secrets.json", "path to the flie that contains secrets used by the application")
	sheetID      = flag.String("sheet_id", "1SgvhTBeVdyTMrCR0wZixO3O0lErh4vqX0--nBpSfYT8", "the id of the puzzle tracking sheet to use")
	puzzlesTab   = flag.String("puzzles_tab", "Puzzle List", "the name of the puzzles tab in the puzzle tracking sheet")
	roundsTab    = flag.String("rounds_tab", "Round Information", "the name of the rounds tab in the puzzle tracking sheet")
	rootFolderID = flag.String("root_folder_id", "1Mp8e1Sd7YXBwcgil62YCgslbQ6twmBlU", "the id of the google drive folder for this year")
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

	dis, err := discord.New(dg, discord.Config{
		QMChannelName:      "qm",
		GeneralChannelName: "whats-going-on",
		TechChannelName:    "tech",
		SolvedCategoryName: "solved",
		PuzzleCategoryName: "puzzles",
		QMRoleName:         "QM",
	})
	if err != nil {
		log.Fatalf("error creating discord client: %v", err)
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

	d, err := drive.New(ctx, *sheetID, *puzzlesTab, *roundsTab, *rootFolderID)
	if err != nil {
		log.Fatalf("error creating drive integration: %v", err)
	}
	h := huntbot.New(dis, d, huntbot.Config{MinWarningFrequency: 10 * time.Minute, InitialWarningDelay: time.Minute})

	log.Print("press ctrl+C to exit")
	dis.RegisterNewMessageHandler("emoji generator", emojiname.Handler)
	dis.RegisterNewMessageHandler("isithuntyet?", huntyet.Handler)
	dis.RegisterNewMessageHandler("bot control", h.ControlHandler)
	dis.RegisterNewMessageHandler("qm manager", dis.QMHandler)
	dis.RegisterNewMessageHandler("voice controller", dis.VoiceChannelHandler)

	go h.WatchSheet(ctx)

	<-ctx.Done()
}

func init() {
	flag.Parse()
}
