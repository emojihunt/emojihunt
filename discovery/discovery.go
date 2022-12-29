package discovery

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/schema"
	"github.com/emojihunt/emojihunt/state"
	"golang.org/x/net/html"
)

func (d *Poller) Scrape(ctx context.Context) ([]*DiscoveredPuzzle, error) {
	// Download
	req, err := http.NewRequestWithContext(ctx, "GET", d.puzzlesURL.String(), nil)
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

	// Parse round structure
	var discovered [][2]*html.Node
	root, err := html.Parse(res.Body)
	if err != nil {
		return nil, err
	}

	if d.groupMode {
		groups := d.groupSelector.MatchAll(root)
		if len(groups) == 0 {
			return nil, fmt.Errorf("no groups found")
		}

		for _, group := range groups {
			nameNode := d.roundNameSelector.MatchFirst(group)
			if nameNode == nil {
				return nil, fmt.Errorf("round name node not found in group: %#v", group)
			}

			puzzleListNode := d.puzzleListSelector.MatchFirst(group)
			if puzzleListNode == nil {
				return nil, fmt.Errorf("puzzle list node not found in group: %#v", group)
			}
			discovered = append(discovered, [2]*html.Node{nameNode, puzzleListNode})
		}
	} else {
		container := d.groupSelector.MatchFirst(root)
		if container == nil {
			return nil, fmt.Errorf("container not found, did login succeed?")
		}

		node := container.FirstChild
		for {
			if d.roundNameSelector.Match(node) {
				nameNode := node

				node = node.NextSibling
				for node.Type == html.TextNode {
					// Skip over text nodes.
					node = node.NextSibling
				}
				if d.puzzleListSelector.Match(node) {
					// Puzzle list found!
					discovered = append(discovered, [2]*html.Node{nameNode, node})
				} else if d.roundNameSelector.Match(node) {
					// Another round heading! This is probably a sub-round;
					// start over treating the new heading as the round name.
					continue
				} else {
					// Unknown structure, abort.
					return nil, fmt.Errorf("puzzle table not found, got: %#v", node)
				}
			}

			// Advance to next node.
			node = node.NextSibling
			if node == nil {
				break
			}
		}

		if len(discovered) == 0 {
			return nil, fmt.Errorf("no rounds found in container: %#v", container)
		}
	}

	// Parse out individual puzzles
	var puzzles []*DiscoveredPuzzle
	for _, pair := range discovered {
		nameNode, puzzleListNode := pair[0], pair[1]
		var roundBuf bytes.Buffer
		collectText(nameNode, &roundBuf)
		roundName := strings.TrimSpace(roundBuf.String())

		puzzleItemNodes := d.puzzleItemSelector.MatchAll(puzzleListNode)
		if len(puzzleItemNodes) == 0 {
			return nil, fmt.Errorf("no puzzle item nodes found in puzzle list: %#v", puzzleListNode)
		}
		for _, item := range puzzleItemNodes {
			var puzzleBuf bytes.Buffer
			collectText(item, &puzzleBuf)

			var u *url.URL
			for _, attr := range item.Attr {
				if attr.Key == "href" {
					u, err = url.Parse(attr.Val)
					if err != nil {
						return nil, fmt.Errorf("invalid puzzle url: %#v", u)
					}
				}
			}
			if u == nil {
				return nil, fmt.Errorf("could not find puzzle url for puzzle: %#v", item)
			}

			puzzles = append(puzzles, &DiscoveredPuzzle{
				Name:  strings.TrimSpace(puzzleBuf.String()),
				URL:   d.puzzlesURL.ResolveReference(u),
				Round: roundName,
			})
		}
	}
	return puzzles, nil
}

func (d *Poller) SyncPuzzles(ctx context.Context, puzzles []*DiscoveredPuzzle) error {
	puzzleMap := make(map[string]*DiscoveredPuzzle)
	for _, puzzle := range puzzles {
		puzzleMap[puzzle.URL.String()] = puzzle
	}

	// Filter out known puzzles; add remaining puzzles
	fragments, rounds, err := d.airtable.ListPuzzleFragmentsAndRounds()
	if err != nil {
		return err
	}

	var newPuzzles []schema.NewPuzzle
	skippedRounds := make(map[string][]string)
	for _, puzzle := range puzzleMap {
		if fragments[strings.ToUpper(puzzle.URL.String())] ||
			fragments[strings.ToUpper(puzzle.Name)] {
			// skip if name or URL matches an existing puzzle
			continue
		}
		round, ok := rounds[puzzle.Round]
		if !ok {
			skippedRounds[puzzle.Round] = append(skippedRounds[puzzle.Round], puzzle.Name)
			continue
		}
		log.Printf("discovery: preparing to add puzzle %q (%s) in round %q", puzzle.Name, puzzle.URL.String(), puzzle.Round)
		newPuzzles = append(newPuzzles, schema.NewPuzzle{
			Name:      puzzle.Name,
			Round:     round,
			PuzzleURL: puzzle.URL.String(),
		})
	}

	if len(newPuzzles) == 0 {
		return nil
	}

	msg := "```\n*** 🧐 NEW PUZZLES ***\n\n"
	for i, puzzle := range newPuzzles {
		if i == newPuzzleLimit {
			msg += fmt.Sprintf("(...and more, %d in total...)\n\n", len(newPuzzles))
			break
		}
		msg += fmt.Sprintf("%s %s\n%s\n\n", puzzle.Round.Emoji, puzzle.Name, puzzle.PuzzleURL)
	}

	var paused bool
	if len(newPuzzles) > newPuzzleLimit {
		paused = true
		msg += fmt.Sprintf(
			"💥 Too many puzzles! Stopped for safety, please contact #%s.\n",
			d.discord.TechChannel.Name,
		)
	} else {
		msg += "Reminder: use `/huntbot kill` to stop the bot.\n"
	}
	msg += "```\n"

	_, err = d.discord.ChannelSend(d.discord.QMChannel, msg)
	if err != nil {
		return err
	} else if paused {
		return nil
	}

	// Warning! Puzzle locks are acquired here and must be released before this
	// function returns.
	created, err := d.airtable.AddPuzzles(newPuzzles)
	if err != nil {
		return err
	}

	time.Sleep(preCreationPause)

	var errs []error
	for _, puzzle := range created {
		if d.state.IsKilled() {
			errs = append(errs, fmt.Errorf("huntbot is disabled"))
		} else {
			if _, err := d.syncer.ForceUpdate(ctx, &puzzle); err != nil {
				errs = append(errs, err)
			}
		}
		puzzle.Unlock()
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors sending new puzzle notifications: %#v", spew.Sdump(errs))
	}

	errs = make([]error, 0)
	for name, puzzles := range skippedRounds {
		if err := d.notifyNewRound(name, puzzles); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("errors sending new round notifications: %#v", spew.Sdump(errs))
	}
	return nil
}

func (d *Poller) notifyNewRound(name string, puzzles []string) error {
	d.state.Lock()
	defer d.state.CommitAndUnlock()

	if _, ok := d.state.DiscoveryNewRounds[name]; ok {
		return nil
	}

	msg := fmt.Sprintf("**:interrobang: New Round: \"%s\"**\n```", name)
	for _, puzzle := range puzzles {
		msg += fmt.Sprintf("%s\n", puzzle)
	}
	msg += "```"

	id, err := d.discord.ChannelSend(d.discord.QMChannel, msg)
	if err != nil {
		return err
	}

	d.state.DiscoveryNewRounds[name] = state.NewRound{CreationMessage: id, SecondsLeft: -1}
	return nil
}

func collectText(n *html.Node, buf *bytes.Buffer) {
	// https://stackoverflow.com/a/18275336
	if n.Type == html.TextNode {
		buf.WriteString(n.Data)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		collectText(c, buf)
	}
}
