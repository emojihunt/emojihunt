package main

import (
	"fmt"

	"github.com/emojihunt/emojihunt/discovery"
)

// This tool can help you out when you're adjusting the CSS selectors for a new
// hunt. Rin `go run ./publisher/cmd` in the root of the project to scrape the
// given URL and print the puzzles that were found.
func main() {
	disc := discovery.New(nil, nil, nil, &discovery.DiscoveryConfig{
		CookieName:  "dummyCookie",
		CookieValue: "dummyValue",
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
