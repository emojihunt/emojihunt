package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/emojihunt/emojihunt/discovery"
)

var (
	config_file = flag.String("config_file", "config.json", "path to the file that contains config used by the application")
)

// This tool can help you out when you're adjusting the CSS selectors for a new
// hunt. Rin `go run ./publisher/cmd` in the root of the project to scrape the
// given URL and print the puzzles that were found.
func main() {
	bs, err := os.ReadFile(*config_file)
	if err != nil {
		log.Fatalf("error opening config.json: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(bs, &config); err != nil {
		log.Fatalf("error parsing config.json: %v", err)
	}

	discoveryConfig := config["autodiscovery"].(map[string]interface{})
	disc := discovery.New(nil, nil, nil, &discovery.DiscoveryConfig{
		CookieName:  discoveryConfig["cookie_name"].(string),
		CookieValue: discoveryConfig["cookie_value"].(string),
	}, nil)
	puzzles, err := disc.Scrape()
	if err != nil {
		fmt.Printf("fatal error: %#v\n", err)
		return
	}

	currentRound := ""
	for _, puzzle := range puzzles {
		if puzzle.Round != currentRound {
			fmt.Printf("\nRound: %s\n", puzzle.Round)
			currentRound = puzzle.Round
		}
		fmt.Printf(" - %s\t(%s)\n", puzzle.Name, puzzle.URL.String())
	}
}

func init() {
	flag.Parse()
}
