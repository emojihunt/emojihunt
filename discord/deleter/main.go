package main

import (
	"flag"
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
	"golang.org/x/xerrors"
)

var (
	prod     = flag.Bool("prod", false, "selects development or production")
	category = flag.String("category", "", "name of category to delete from")
	dryRun   = flag.Bool("dry_run", true, "whether to run in dry run mode or not")
)

func main() {
	// Create Discord client and get channel list
	token, ok := os.LookupEnv("DISCORD_TOKEN")
	if !ok {
		panic("DISCORD_TOKEN is required")
	}
	dg, err := discordgo.New(token)
	if err != nil {
		panic(err)
	}
	err = dg.Open()
	defer dg.Close()
	if err != nil {
		panic(err)
	}
	guildID := discord.DevConfig.GuildID
	if *prod {
		guildID = discord.ProdConfig.GuildID
	}
	chs, err := dg.GuildChannels(guildID)
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
		panic(xerrors.Errorf("could not find category %q", *category))
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
