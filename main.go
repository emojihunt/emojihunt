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
	"github.com/gauravjsingh/emojihunt/bot"
	"github.com/gauravjsingh/emojihunt/client"
	"github.com/gauravjsingh/emojihunt/huntbot"
	"github.com/gauravjsingh/emojihunt/server"
	"github.com/gauravjsingh/emojihunt/syncer"
)

var (
	secretsFile  = flag.String("secrets_file", "secrets.json", "path to the flie that contains secrets used by the application")
	rootFolderID = flag.String("root_folder_id", "1Mp8e1Sd7YXBwcgil62YCgslbQ6twmBlU", "the id of the google drive folder for this year")
	guildID      = flag.String("discord_guild_id", "793599987694436374", "the id of the discord guild")
	baseID       = flag.String("airtable_base_id", "appmjhGfZLui26Xow", "the id of the airtable base")
	tableName    = flag.String("airtable_table_name", "Puzzle Tracker", "the name of the table in the airtable base")
	certFile     = flag.String("certificate", "/etc/letsencrypt/live/huntbox.emojihunt.tech/fullchain.pem", "the path to the server certificate")
	keyFile      = flag.String("private_key", "/etc/letsencrypt/live/huntbox.emojihunt.tech/privkey.pem", "the path to the server private key")
)

type secrets struct {
	AirtableToken string `json:"airtable_token"`
	DiscordToken  string `json:"discord_token"`
	HuntboxToken  string `json:"huntbox_token"`
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
		log.Fatalf("error creating discordgo client: %v", err)
	}

	err = dg.Open()
	defer dg.Close()
	if err != nil {
		log.Fatalf("error opening discord connection: %v", err)
	}

	dis, err := client.NewDiscord(dg, client.DiscordConfig{
		GuildID:            *guildID,
		QMChannelName:      "qm",
		GeneralChannelName: "whats-going-on",
		TechChannelName:    "tech",
		SolvedCategoryName: "Solved",
		PuzzleCategoryName: "Puzzles",
		QMRoleName:         "QM",
	})
	if err != nil {
		log.Fatalf("error creating discord client: %v", err)
	}

	air := client.NewAirtable(secrets.AirtableToken, *baseID, *tableName)

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

	d, err := client.NewDrive(ctx, *rootFolderID)
	if err != nil {
		log.Fatalf("error creating drive integration: %v", err)
	}
	syn := syncer.New(air, dis, d)
	hunt := huntbot.New(air, dis, syn, huntbot.Config{MinWarningFrequency: 10 * time.Minute, InitialWarningDelay: time.Minute})

	log.Print("press ctrl+C to exit")
	dis.RegisterNewMessageHandler("emoji generator", bot.MakeEmojiNameHandler())
	dis.RegisterNewMessageHandler("isithuntyet?", bot.MakeHuntYetHandler())
	dis.RegisterNewMessageHandler("bot control", hunt.Handler)
	dis.RegisterNewMessageHandler("qm manager", bot.MakeQMHandler(dis))
	dis.RegisterNewMessageHandler("voice channel helper", bot.MakeVoiceRoomHandler(air, dis))

	go hunt.PollDatabase(ctx)

	server := server.New(air, dis, d, secrets.HuntboxToken)
	server.Start(*certFile, *keyFile)

	<-ctx.Done()
}

func init() {
	flag.Parse()
}
