package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/emojihunt/emojihunt/discovery"
)

var (
	config_path = flag.String("config", "config.json", "path to the configuration file")
)

// This tool can help you out when you're adjusting the CSS selectors for a new
// hunt. Run `go run ./discovery/checker` in the root of the project to scrape
// the given URL and print the puzzles that were found.
func main() {
	bs, err := os.ReadFile(*config_path)
	if err != nil {
		log.Panicf("error opening config.json: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(bs, &raw); err != nil {
		log.Panicf("error parsing config.json: %v", err)
	}
	node, err := json.Marshal(raw["autodiscovery"])
	if err != nil {
		log.Panicf("error navigating config.json: %v", err)
	}
	config := discovery.DiscoveryConfig{}
	if err := json.Unmarshal(node, &config); err != nil {
		log.Panicf("error parsing autodiscovery node: %v", err)
	}

	disc := discovery.New(nil, nil, nil, &config, nil)
	puzzles, err := disc.Scrape(context.Background())
	if err != nil {
		fmt.Printf("fatal error: %#v\n", err)
		return
	}

	currentRound := ""
	for _, puzzle := range puzzles {
		if puzzle.Round.Name == "" {
			panic("blank round name")
		}
		if puzzle.Round.Name != currentRound {
			if currentRound != "" {
				fmt.Println()
			}
			fmt.Printf("Round: \"%s\"\n", puzzle.Round.Name)
			currentRound = puzzle.Round.Name
		}
		if len(puzzle.Name) <= 32 {
			fmt.Printf(" - %-32s  %s\n", puzzle.Name, puzzle.PuzzleURL)
		} else {
			fmt.Printf(" - %s\n   %s\n", puzzle.Name, puzzle.PuzzleURL)
		}
	}
}

func init() {
	flag.Parse()
}
