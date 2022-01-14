package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

	"github.com/bwmarrin/discordgo"
)

var (
	config_file = flag.String("config_file", "config.json", "path to the file that contains config used by the application")
	category    = flag.String("category", "", "name of category to delete from")
	dryRun      = flag.Bool("dry_run", true, "whether to run in dry run mode or not")
)

func main() {
	bs, err := ioutil.ReadFile(*config_file)
	if err != nil {
		log.Fatalf("error opening config.json: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(bs, &config); err != nil {
		log.Fatalf("error parsing config.json: %v", err)
	}

	discordConfig := config["discord"].(map[string]interface{})

	dg, err := discordgo.New(discordConfig["auth_token"].(string))
	if err != nil {
		log.Fatalf("error creating discordgo client: %v", err)
	}

	err = dg.Open()
	defer dg.Close()
	if err != nil {
		log.Fatalf("error opening discord connection: %v", err)
	}

	chs, err := dg.GuildChannels(discordConfig["guild_id"].(string))
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
