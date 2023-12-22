package main

import (
	"context"
	"flag"
	"fmt"

	"github.com/emojihunt/emojihunt/discovery"
)

// This tool can help you out when you're adjusting the CSS selectors for a new
// hunt. Run `go run ./discovery/checker` in the root of the project to scrape
// the given URL and print the puzzles that were found.
func main() {
	// Run discovery with stubbed-out discovery client
	disc := discovery.New(context.Background(), nil, nil, nil)
	puzzles, err := disc.Scrape(context.Background())
	if err != nil {
		panic(err)
	}

	// Print results
	currentRound := ""
	for _, puzzle := range puzzles {
		if puzzle.RoundName == "" {
			panic("blank round name")
		}
		if puzzle.RoundName != currentRound {
			if currentRound != "" {
				fmt.Println()
			}
			fmt.Printf("Round: \"%s\"\n", puzzle.RoundName)
			currentRound = puzzle.RoundName
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
