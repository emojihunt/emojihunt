package discovery

import (
	"bytes"
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/emojihunt/emojihunt/state"
	"golang.org/x/net/html"
	"golang.org/x/xerrors"
)

func (p *Poller) Scrape(ctx context.Context) ([]state.DiscoveredPuzzle, error) {
	// Download
	req, err := http.NewRequestWithContext(ctx, "GET", p.puzzlesURL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.AddCookie(p.cookie)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	} else if res.StatusCode != http.StatusOK {
		return nil, xerrors.Errorf("failed to fetch puzzle list: status code %v", res.Status)
	}

	// Parse round structure
	var discovered [][2]*html.Node
	root, err := html.Parse(res.Body)
	if err != nil {
		return nil, err
	}

	if p.groupMode {
		groups := p.groupSelector.MatchAll(root)
		if len(groups) == 0 {
			return nil, xerrors.Errorf("no groups found")
		}

		for _, group := range groups {
			nameNode := p.roundNameSelector.MatchFirst(group)
			if nameNode == nil {
				return nil, xerrors.Errorf("round name node not found in group: %#v", group)
			}

			puzzleListNode := p.puzzleListSelector.MatchFirst(group)
			if puzzleListNode == nil {
				return nil, xerrors.Errorf("puzzle list node not found in group: %#v", group)
			}
			discovered = append(discovered, [2]*html.Node{nameNode, puzzleListNode})
		}
	} else {
		container := p.groupSelector.MatchFirst(root)
		if container == nil {
			return nil, xerrors.Errorf("container not found, did login succeed?")
		}

		node := container.FirstChild
		for {
			if p.roundNameSelector.Match(node) {
				nameNode := node

				node = node.NextSibling
				for node.Type == html.TextNode {
					// Skip over text nodes.
					node = node.NextSibling
				}
				if p.puzzleListSelector.Match(node) {
					// Puzzle list found!
					discovered = append(discovered, [2]*html.Node{nameNode, node})
				} else if p.roundNameSelector.Match(node) {
					// Another round heading! This is probably a sub-round;
					// start over treating the new heading as the round name.
					continue
				} else {
					// Unknown structure, abort.
					return nil, xerrors.Errorf("puzzle table not found, got: %#v", node)
				}
			}

			// Advance to next node.
			node = node.NextSibling
			if node == nil {
				break
			}
		}

		if len(discovered) == 0 {
			return nil, xerrors.Errorf("no rounds found in container: %#v", container)
		}
	}

	// Parse out individual puzzles
	var puzzles []state.DiscoveredPuzzle
	for _, pair := range discovered {
		nameNode, puzzleListNode := pair[0], pair[1]
		var roundBuf bytes.Buffer
		collectText(nameNode, &roundBuf)
		roundName := strings.TrimSpace(roundBuf.String())

		puzzleItemNodes := p.puzzleItemSelector.MatchAll(puzzleListNode)
		if len(puzzleItemNodes) == 0 {
			return nil, xerrors.Errorf("no puzzle item nodes found in puzzle list: %#v", puzzleListNode)
		}
		for _, item := range puzzleItemNodes {
			var puzzleBuf bytes.Buffer
			collectText(item, &puzzleBuf)

			var u *url.URL
			for _, attr := range item.Attr {
				if attr.Key == "href" {
					u, err = url.Parse(attr.Val)
					if err != nil {
						return nil, xerrors.Errorf("invalid puzzle url: %#v", u)
					}
				}
			}
			if u == nil {
				return nil, xerrors.Errorf("could not find puzzle url for puzzle: %#v", item)
			}

			url := p.puzzlesURL.ResolveReference(u).String()
			puzzles = append(puzzles, state.DiscoveredPuzzle{
				Name:  strings.TrimSpace(puzzleBuf.String()),
				Round: roundName,
				URL:   url,
			})
		}
	}
	return puzzles, nil
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
