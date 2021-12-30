package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	"github.com/bwmarrin/discordgo"
)

var (
	secretsFile = flag.String("secrets_file", "secrets.json", "path to the flie that contains secrets used by the application")
	guildID     = flag.String("guild_id", "793599987694436374", "the id of the discord guild")
	category    = flag.String("category", "", "name of category to delete from")
	dryRun      = flag.Bool("dry_run", true, "whether to run in dry run mode or not")
)

func main() {
	bs, err := ioutil.ReadFile(*secretsFile)
	if err != nil {
		log.Fatalf("error opening secrets.json: %v", err)
	}

	var secrets map[string]interface{}
	if err := json.Unmarshal(bs, &secrets); err != nil {
		log.Fatalf("error parsing secrets.json: %v", err)
	}

	dg, err := discordgo.New(secrets["discord_token"].(string))
	if err != nil {
		log.Fatalf("error creating discordgo client: %v", err)
	}

	err = dg.Open()
	defer dg.Close()
	if err != nil {
		log.Fatalf("error opening discord connection: %v", err)
	}

	chs, err := dg.GuildChannels(*guildID)
	if err != nil {
		log.Fatalf("error listing channels: %v", err)
	}

	var categoryID = ""
	log.Printf("Listing Categories")
	for _, ch := range chs {
		if ch.Type == discordgo.ChannelTypeGuildCategory {
			if ch.Name == *category {
				log.Printf(" * %s", ch.Name)
				categoryID = ch.ID
			} else {
				log.Printf(" - %s", ch.Name)
			}
		}
	}
	if categoryID == "" {
		log.Fatalf("Could not find category %q", *category)
	}

	var action = "real"
	if *dryRun {
		action = "dry run"
	}
	log.Printf("Deleting Channels (%s)", action)
	for _, ch := range chs {
		if ch.ParentID == categoryID {
			log.Printf(" * %s", ch.Name)
			if !*dryRun {
				dg.ChannelDelete(ch.ID)
			}
		}
	}

	log.Printf(" * %s", *category)
	if !*dryRun {
		dg.ChannelDelete(categoryID)
	}

	log.Printf("Done!")
}

func init() {
	flag.Parse()
}
