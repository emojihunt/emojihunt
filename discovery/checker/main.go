package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
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
	// Load config file
	bs, err := os.ReadFile(*config_path)
	if err != nil {
		panic(err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(bs, &raw); err != nil {
		panic(err)
	}
	node, err := json.Marshal(raw["autodiscovery"])
	if err != nil {
		panic(err)
	}
	config := discovery.DiscoveryConfig{}
	if err := json.Unmarshal(node, &config); err != nil {
		panic(err)
	}

	// Run discovery with stubbed-out discovery client
	disc := discovery.New(context.Background(), nil, nil, nil, &config, nil)
	puzzles, err := disc.Scrape(context.Background())
	if err != nil {
		panic(err)
	}

	// Print results
	currentRound := ""
	for _, puzzle := range puzzles {
		if puzzle.Round == "" {
			panic("blank round name")
		}
		if puzzle.Round != currentRound {
			if currentRound != "" {
				fmt.Println()
			}
			fmt.Printf("Round: \"%s\"\n", puzzle.Round)
			currentRound = puzzle.Round
		}
		if len(puzzle.Name) <= 32 {
			fmt.Printf(" - %-32s  %s\n", puzzle.Name, puzzle.URL)
		} else {
			fmt.Printf(" - %s\n   %s\n", puzzle.Name, puzzle.URL)
		}
	}
}

func init() {
	flag.Parse()
}
