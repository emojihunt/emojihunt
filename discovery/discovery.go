package discovery

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/davecgh/go-spew/spew"
	"github.com/emojihunt/emojihunt/client"
	"github.com/emojihunt/emojihunt/schema"
	"github.com/emojihunt/emojihunt/state"
	"golang.org/x/net/html"
)

func (d *Poller) Scrape() ([]*DiscoveredPuzzle, error) {
	// Download
	req, err := http.NewRequest("GET", d.puzzlesURL.String(), nil)
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

func (d *Poller) SyncPuzzles(puzzles []*DiscoveredPuzzle) error {
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
			log.Printf("discovery: skipping puzzle %q due to unknown round %q", puzzle.Name, puzzle.Round)
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

	if len(newPuzzles) > newPuzzleLimit {
		return fmt.Errorf("too many new puzzles; aborting for safety (%d)", len(newPuzzles))
	}

	created, err := d.airtable.AddPuzzles(newPuzzles)
	if err != nil {
		return err
	}

	var errs []error
	for _, puzzle := range created {
		if err := d.notifyNewPuzzle(&puzzle); err != nil {
			errs = append(errs, err)
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

func (d *Poller) RegisterApproveCommand(ctx context.Context, discord *client.Discord) {
	command := &client.DiscordCommand{
		InteractionType: discordgo.InteractionMessageComponent,
		CustomID:        "discovery.approve",
		OnlyOnce:        true,
		Async:           true,
		Handler: func(s *discordgo.Session, i *client.DiscordCommandInput) (string, error) {
			parts := strings.Split(i.Command, "/")
			if len(parts) < 2 {
				return "", fmt.Errorf("could not parse Airtable ID from command: %q", i.Command)
			}
			puzzle, err := d.airtable.LockByID(parts[1])
			if err != nil {
				return "", err
			}
			defer puzzle.Unlock()

			if !puzzle.Pending {
				return fmt.Sprintf(":man_shrugging: Puzzle %q is already approved, %s!", puzzle.Name, i.User.Mention()), nil
			}

			if _, err := d.syncer.ForceUpdate(ctx, puzzle); err != nil {
				return "", err
			}
			return fmt.Sprintf(":ok_hand: I've created puzzle %q, %s!", puzzle.Name, i.User.Mention()), nil
		},
	}
	discord.AddCommand(command)
}

func (d *Poller) notifyNewPuzzle(puzzle *schema.Puzzle) error {
	msg := fmt.Sprintf(
		"**%s New puzzle detected!** Name: %q, Round: %s, URL: <%s>",
		puzzle.Rounds.Emojis(), puzzle.Name, puzzle.Rounds.Names(), puzzle.PuzzleURL,
	)
	components := []discordgo.MessageComponent{
		discordgo.Button{
			Label: "Edit in Airtable",
			Style: discordgo.LinkButton,
			Emoji: discordgo.ComponentEmoji{Name: "📝"},
			URL:   d.airtable.EditURL(puzzle),
		},
		discordgo.Button{
			Label:    "Approve",
			Style:    discordgo.SuccessButton,
			Emoji:    discordgo.ComponentEmoji{Name: "🔨"},
			CustomID: "discovery.approve/" + puzzle.AirtableRecord.ID,
		},
	}
	_, err := d.discord.ChannelSendComponents(d.discord.QMChannel, msg, components)
	return err
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
