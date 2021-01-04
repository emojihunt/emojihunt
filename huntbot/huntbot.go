package huntbot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/discord"
	"github.com/gauravjsingh/emojihunt/drive"
)

type HuntBot struct {
	dis   *discord.Client
	drive *drive.Drive

	solvedPuzzles map[string]bool // set of names
	mu            sync.Mutex      // hold while accessing solvedPuzzles
}

func New(dis *discord.Client, drive *drive.Drive) *HuntBot {
	return &HuntBot{dis: dis, drive: drive}
}

func (h *HuntBot) notifyNewPuzzle(name, puzzleURL, sheetURL, channelID string) error {
	log.Printf("Posting information about new puzzle %q", name)
	// TODO: also edit sheet to link to channel/puzzle

	// Pin a message with the spreadsheet URL to the channel
	if err := h.dis.ChannelSendAndPin(channelID, fmt.Sprintf("Spreadsheet: %s\nPuzzle: %s", sheetURL, puzzleURL)); err != nil {
		return fmt.Errorf("error pinning puzzle info: %v", err)
	}

	// Post a message in the general channel with a link to the puzzle.
	if err := h.dis.GeneralChannelSend(fmt.Sprintf("There is a new puzzle %s!\nPuzzle URL: %s\nChannel <#%s>", name, puzzleURL, channelID)); err != nil {
		return fmt.Errorf("error posting new puzzle announcement: %v", err)
	}

	return nil
}

func (h *HuntBot) NewPuzzle(ctx context.Context, name string) error {
	id, err := h.dis.CreateChannel(name)
	if err != nil {
		return fmt.Errorf("error creating discord channel for %q: %v", name, err)
	}
	// Create Spreadsheet
	sheetURL, err := h.drive.CreateSheet(ctx, name, "Unknown Round") // TODO
	if err != nil {
		return fmt.Errorf("error creating spreadsheet for %q: %v", name, err)
	}

	// If via bot, also take puzzle url as a param
	puzzleURL := "https://en.wikipedia.org/wiki/Main_Page"

	return h.notifyNewPuzzle(name, puzzleURL, sheetURL, id)
}

func (h *HuntBot) NewPuzzleHandler(s *discordgo.Session, m *discordgo.MessageCreate) error {
	if m.Author.ID == s.State.User.ID || !strings.HasPrefix(m.Content, "!newpuzzle") {
		return nil
	}

	parts := strings.Split(m.Content, " ")
	if len(parts) < 2 {
		// send a bad usage message to the channel
		return fmt.Errorf("not able to find new puzzle name from %q", m.Content)
	}
	if err := h.NewPuzzle(context.Background(), parts[1]); err != nil {
		return fmt.Errorf("error creating puzzle: %v", err)
	}
	return nil
}

func (h *HuntBot) maybeMarkSolved(ctx context.Context, puzzle drive.PuzzleInfo) error {
	h.mu.Lock()
	defer h.mu.Unlock() // TODO: finer locking
	if h.solvedPuzzles[puzzle.Name] {
		// already marked this one solved
		return nil
	}

	log.Printf("Archiving channel for %q", puzzle.Name)
	channelID, err := h.dis.ChannelID(puzzle.DiscordURL)
	if err != nil {
		return err
	}

	err = h.dis.ArchiveChannel(channelID)
	if errors.Is(err, discord.ChannelNotFound) {
		// already archived
	} else if err != nil {
		return fmt.Errorf("unable to archive channel for %q: %v", puzzle.Name, err)
	}

	err = h.drive.MarkSheetSolved(ctx, puzzle.DocURL)
	if err != nil {
		return err
	}

	if err := h.dis.QMChannelSend(fmt.Sprintf("Puzzle %q was solved!", puzzle.Name)); err != nil {
		return fmt.Errorf("error posting new puzzle announcement: %v", err)
	}

	h.solvedPuzzles[puzzle.Name] = true
	return nil
}

func (h *HuntBot) pollAndUpdate(ctx context.Context) error {
	puzzles, err := h.drive.ReadFullSheet()
	if err != nil {
		return err
	}

	for _, puzzle := range puzzles {
		if puzzle.Name != "" && puzzle.PuzzleURL != "" && puzzle.Round.Name != "" {
			// TODO: warn if puzzle.Name is set but others haven't been for a
			// while?
			requiresUpdate := false
			if puzzle.DocURL == "" {
				puzzle.DocURL, err = h.drive.CreateSheet(ctx, puzzle.Name, puzzle.Round.Name)
				if err != nil {
					return fmt.Errorf("error creating spreadsheet for %q: %v", puzzle.Name, err)
				}
				requiresUpdate = true
			}

			if puzzle.DiscordURL == "" {
				log.Printf("Adding channel for new puzzle %q", puzzle.Name)
				id, err := h.dis.CreateChannel(puzzle.Name)
				if err != nil {
					return fmt.Errorf("error creating discord channel for %q: %v", puzzle.Name, err)
				}

				puzzle.DiscordURL = h.dis.ChannelURL(id)

				// Treat discord URL as the sentinel to also notify everyone
				if err := h.notifyNewPuzzle(puzzle.Name, puzzle.PuzzleURL, puzzle.DocURL, id); err != nil {
					return fmt.Errorf("error notifying channel about new puzzle %q: %v", puzzle.Name, err)
				}
				if err := h.drive.SetDiscordURL(ctx, puzzle); err != nil {
					return fmt.Errorf("error setting discord URL for puzzle %q: %v", puzzle.Name, err)
				}
				requiresUpdate = true
			}
			if requiresUpdate {
				if err := h.drive.UpdatePuzzle(ctx, puzzle); err != nil {
					return fmt.Errorf("error updating sheet info for puzzle %q: %v", puzzle.Name, err)
				}
			}
		}

		if puzzle.Answer != "" && puzzle.Status == drive.Solved {
			err := h.maybeMarkSolved(ctx, puzzle)
			if err != nil {
				return fmt.Errorf("failed to mark puzzle %q solved: %v", puzzle.Name, err)
			}
		}
	}

	return nil
}

func (h *HuntBot) WatchSheet(ctx context.Context) {
	// we don't have a way to subscribe to updates, so we just poll the sheet
	// TODO: if sheet last-mod is since our last run, noop
	failures := 0
	for {
		err := h.pollAndUpdate(ctx)
		if err != nil {
			// log always, but ping after 3 consecutive failures, then every 10, to avoid spam
			log.Printf("watching sheet failed: %v", err)
			failures++
			if failures%10 == 3 {
				h.dis.TechChannelSend(fmt.Sprintf("watching sheet failed: %v", err))
			}
		} else {
			failures = 0
		}

		select {
		case <-ctx.Done():
			log.Print("exiting watcher due to signal")
			return
		case <-time.After(10 * time.Second):
		}
	}
}
