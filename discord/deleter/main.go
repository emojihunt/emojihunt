package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
)

var (
	configPath = flag.String("config", "config.json", "path to the configuration file")
	category   = flag.String("category", "", "name of category to delete from")
	dryRun     = flag.Bool("dry_run", true, "whether to run in dry run mode or not")
)

func main() {
	// Load config file
	bs, err := os.ReadFile(*configPath)
	if err != nil {
		panic(err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(bs, &raw); err != nil {
		panic(err)
	}
	config := raw["discord"].(map[string]interface{})

	// Create Discord client and get channel list
	dg, err := discordgo.New(config["auth_token"].(string))
	if err != nil {
		panic(err)
	}
	err = dg.Open()
	defer dg.Close()
	if err != nil {
		panic(err)
	}
	chs, err := dg.GuildChannels(config["guild_id"].(string))
	if err != nil {
		panic(err)
	}

	// Print results
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
		panic(fmt.Errorf("could not find category %q", *category))
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
