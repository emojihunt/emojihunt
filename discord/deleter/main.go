package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/emojihunt/emojihunt/discord"
)

var (
	prod = flag.Bool("prod", false, "selects development or production")
)

func init() {
	flag.Parse()
}

func main() {
	var reader = bufio.NewReader(os.Stdin)
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
	var cfg = discord.DevConfig
	if *prod {
		cfg = discord.ProdConfig
	}
	chs, err := dg.GuildChannels(cfg.GuildID)
	if err != nil {
		panic(err)
	}

	for _, ch := range chs {
		if ch.Type != discordgo.ChannelTypeGuildCategory {
			continue
		}
		fmt.Printf("category %q\n", ch.Name)
		if ch.ID == cfg.TeamCategoryID {
			fmt.Printf(" - skip (team category)\n")
			continue
		}
		var ntext, nvoice int
		for _, c := range chs {
			if c.ParentID != ch.ID {
				continue
			}
			switch c.Type {
			case discordgo.ChannelTypeGuildText:
				ntext += 1
			case discordgo.ChannelTypeGuildVoice:
				nvoice += 1
			}
		}
		if nvoice > 0 {
			fmt.Printf(" - skip (voice category)\n")
			continue
		}
		var solved = strings.Contains(ch.Name, "Solved")
		if solved && ntext == 0 {
			fmt.Printf(" - skip (empty)\n")
			continue
		} else if solved {
			fmt.Printf(" - clear? [y/n] ")
		} else {
			fmt.Printf(" - clear & delete? [y/n] ")
		}
		ans, err := reader.ReadString('\n')
		if err != nil {
			panic(err)
		} else if strings.TrimSpace(strings.ToLower(ans)) != "y" {
			fmt.Printf(" - skip\n")
			continue
		}
		for _, c := range chs {
			if c.ParentID != ch.ID {
				continue
			}
			fmt.Printf(" - delete channel %q\n", c.Name)
			_, err = dg.ChannelDelete(c.ID)
			if err != nil {
				panic(err)
			}
		}
		if !solved {
			fmt.Printf(" - delete category %q\n", ch.Name)
			_, err = dg.ChannelDelete(ch.ID)
			if err != nil {
				panic(err)
			}
		}
	}
}
