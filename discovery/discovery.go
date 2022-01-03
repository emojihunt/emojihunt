package discovery

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/andybalholm/cascadia"
	"github.com/gauravjsingh/emojihunt/schema"
	"golang.org/x/net/html"
)

var (
	// URL of the "All Puzzles" page on the hunt website
	puzzleListURL, _ = url.Parse("http://puzzles.mit.edu/2021/puzzles.html")

	// URL of the Websocket endpoint
	websocketURL    *url.URL = nil
	websocketOrigin          = "" // fmt.Sprintf("https://%s/", websocketURL.Host)

	// A "group" is the HTML element that contains the round name and all of the
	// puzzles in that round. This is a CSS selector that matches all of the
	// groups in the page. This selector is run on the root of the document.
	groupSelector = cascadia.MustCompile(".info div section")

	// A CSS selector that matches the round name element, i.e. an element whose
	// contents are the name of the round. This selector is run on each group.
	roundNameSelector = cascadia.MustCompile("a h3")

	// A CSS selector that matches each of the puzzles in the group. This
	// selector is expected to match <a> tags where the "href" attribute is the
	// puzzle URL and the immediate contents of the tag are the name of the
	// puzzle. This selector is run on each group.
	puzzleSelector = cascadia.MustCompile("td a")
)

// EXAMPLES
//
// 2021 (http://puzzles.mit.edu/2021/puzzles.html)
// - Group:      `.info div section`
// - Round Name: `a h3`
// - Puzzle:     `td a`
//
// 2020 (http://puzzles.mit.edu/2020/puzzles/)
// - Group:      `#loplist > li`
// - Round Name: `a`
// - Puzzle:     `ul li a`
//

func (d *Poller) Scrape() ([]*DiscoveredPuzzle, error) {
	// Download
	req, err := http.NewRequest("GET", puzzleListURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.AddCookie(d.cookie)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	} else if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch puzzle list: status code %v", res.Status)
	}

	// Parse
	var puzzles []*DiscoveredPuzzle
	root, err := html.Parse(res.Body)
	if err != nil {
		return nil, err
	}

	groups := groupSelector.MatchAll(root)
	if len(groups) < 1 {
		return nil, fmt.Errorf("failed to parse puzzle list: no groups found")
	}

	for _, group := range groups {
		nameNode := roundNameSelector.MatchFirst(group)
		if nameNode == nil {
			return nil, fmt.Errorf("round name not found for group: %#v", group)
		}
		roundName := strings.TrimSpace(nameNode.FirstChild.Data)

		puzzleNodes := puzzleSelector.MatchAll(group)
		if len(puzzleNodes) < 1 {
			return nil, fmt.Errorf("no puzzles found for group: %#v", group)
		}
		for _, puzzleNode := range puzzleNodes {
			var u *url.URL
			for _, attr := range puzzleNode.Attr {
				if attr.Key == "href" {
					u, err = url.Parse(attr.Val)
					if err != nil {
						return nil, fmt.Errorf("invalid puzzle url: %#v", u)
					}
				}
			}
			if u == nil {
				return nil, fmt.Errorf("could not find puzzle url for puzzle: %#v", puzzleNode)
			}
			puzzles = append(puzzles, &DiscoveredPuzzle{
				Name:  strings.TrimSpace(puzzleNode.FirstChild.Data),
				URL:   puzzleListURL.ResolveReference(u),
				Round: roundName,
			})
		}
	}
	return puzzles, nil
}

func (d *Poller) SyncPuzzles(puzzles []*DiscoveredPuzzle) error {
	puzzleMap := make(map[string]*DiscoveredPuzzle)
	for _, puzzle := range puzzles {
		puzzleMap[puzzle.URL.String()] = puzzle
	}

	// Filter out known puzzles
	records, err := d.airtable.ListRecords()
	if err != nil {
		return err
	}

	rounds := make(map[string]schema.Round)
	knownURLs := make(map[string]bool)
	knownNames := make(map[string]bool)
	for _, record := range records {
		rounds[record.Round.Name] = record.Round
		knownURLs[strings.ToUpper(record.PuzzleURL)] = true
		knownURLs[strings.ToUpper(record.OriginalURL)] = true
		knownNames[strings.ToUpper(record.Name)] = true
	}

	// Add remaining puzzles
	var newPuzzles []*schema.NewPuzzle
	skippedRounds := make(map[string]bool)
	for _, puzzle := range puzzleMap {
		if knownURLs[strings.ToUpper(puzzle.URL.String())] ||
			knownNames[strings.ToUpper(puzzle.Name)] {
			// skip if name or URL matches an existing puzzle
			continue
		}
		round, ok := rounds[puzzle.Round]
		if !ok {
			log.Printf("discovery: skipping puzzle %q due to unknown round %q", puzzle.Name, puzzle.Round)
			skippedRounds[puzzle.Round] = true
			continue
		}
		log.Printf("discovery: preparing to add puzzle %q (%s) in round %q", puzzle.Name, puzzle.URL.String(), puzzle.Round)
		newPuzzles = append(newPuzzles, &schema.NewPuzzle{
			Name:      puzzle.Name,
			Round:     round,
			PuzzleURL: puzzle.URL.String(),
		})
	}

	if len(newPuzzles) > newPuzzleLimit {
		return fmt.Errorf("too many new puzzles; aborting for safety (%d)", len(newPuzzles))
	}

	if err := d.airtable.AddPuzzles(newPuzzles); err != nil {
		return err
	}

	return d.notifyNewRounds(skippedRounds)
}

func (d *Poller) notifyNewRounds(rounds map[string]bool) error {
	var array []string
	shouldNotify := false
	for round := range rounds {
		array = append(array, round)
		lastNotified, ok := d.newRounds[round]
		if !ok || time.Since(lastNotified) > roundNotifyFrequency {
			shouldNotify = true
		}
	}
	if !shouldNotify {
		return nil
	}

	msg := fmt.Sprintf(
		"New rounds are available! Please add at least one puzzle from each round to " +
			"Airtable (after that, puzzle auto-discovery can take over). Rounds: " +
			strings.Join(array, ", "),
	)
	if err := d.discord.QMChannelSend(msg); err != nil {
		return err
	}

	for round := range rounds {
		d.newRounds[round] = time.Now()
	}
	return nil
}
