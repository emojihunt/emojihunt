package huntbot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/schema"
)

const pollInterval = 10 * time.Second

func (h *HuntBot) PollDatabase(ctx context.Context) {
	// Airtable doesn't support webhooks, so we have to poll the database for
	// updates.
	failures := 0
	for {
		if h.isEnabled() {
			puzzles, err := h.airtable.ListRecords()
			if err != nil {
				// Log errors always, but ping after 3 consecutive failures,
				// then every 10, to avoid spam
				log.Printf("polling sheet failed: %v", err)
				failures++
				if failures%10 == 3 {
					h.discord.TechChannelSend(fmt.Sprintf("polling sheet failed: %v", err))
				}
			} else {
				failures = 0
			}

			h.mu.Lock()
			h.channelToPuzzle = make(map[string]string)
			for _, puzzle := range puzzles {
				h.channelToPuzzle[puzzle.DiscordChannel] = puzzle.Name
			}
			h.mu.Unlock()

			for _, puzzle := range puzzles {
				err := h.processPuzzle(ctx, &puzzle)
				if err != nil {
					// Log errors and keep going.
					log.Printf("updating puzzle failed: %v", err)
				}
			}
		} else {
			log.Printf("bot disabled, skipping update")
		}

		select {
		case <-ctx.Done():
			log.Print("exiting watcher due to signal")
			return
		case <-time.After(pollInterval):
		}
	}
}

func (h *HuntBot) processPuzzle(ctx context.Context, puzzle *schema.Puzzle) error {
	if puzzle.Name == "" || puzzle.PuzzleURL == "" || puzzle.Round.Name == "" {
		// Occasionally warn the QM about puzzles that are missing fields.
		if puzzle.Name != "" {
			if err := h.warnPuzzle(ctx, puzzle); err != nil {
				return fmt.Errorf("error warning about malformed puzzle %q: %v", puzzle.Name, err)
			}
		}
		return nil
	}

	if puzzle.SpreadsheetID == "" {
		spreadsheet, err := h.drive.CreateSheet(ctx, puzzle.Name, puzzle.Round.Name)
		if err != nil {
			return fmt.Errorf("error creating spreadsheet for %q: %v", puzzle.Name, err)
		}

		puzzle, err = h.airtable.UpdateSpreadsheetID(puzzle, spreadsheet)
		if err != nil {
			return fmt.Errorf("error setting spreadsheet id for puzzle %q: %v", puzzle.Name, err)
		}
	}

	if puzzle.DiscordChannel == "" {
		log.Printf("Adding channel for new puzzle %q", puzzle.Name)
		channel, err := h.discord.CreateChannel(puzzle.Name)
		if err != nil {
			return fmt.Errorf("error creating discord channel for %q: %v", puzzle.Name, err)
		}

		// Treat discord URL as the sentinel to also notify everyone
		if err := h.notifyNewPuzzle(puzzle, puzzle.DiscordChannel); err != nil {
			return fmt.Errorf("error notifying channel about new puzzle %q: %v", puzzle.Name, err)
		}

		puzzle, err = h.airtable.UpdateDiscordChannel(puzzle, channel)
		if err != nil {
			return fmt.Errorf("error setting discord channel for puzzle %q: %v", puzzle.Name, err)
		}
	}

	if h.setPuzzleStatus(puzzle.Name, puzzle.Status) != puzzle.Status ||
		puzzle.Answer != "" && puzzle.Status.IsSolved() && !h.isArchived(puzzle.Name) {
		// (potential) status change
		if puzzle.Status.IsSolved() {
			if err := h.markSolved(ctx, puzzle); err != nil {
				return fmt.Errorf("failed to mark puzzle %q solved: %v", puzzle.Name, err)
			}
		} else {
			if err := h.logStatus(ctx, puzzle); err != nil {
				return fmt.Errorf("failed to mark puzzle %q %v: %v", puzzle.Name, puzzle.Status, err)
			}
		}
	}

	return nil
}

func (h *HuntBot) warnPuzzle(ctx context.Context, puzzle *schema.Puzzle) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if lastWarning, ok := h.lastWarnTime[puzzle.Name]; !ok {
		h.lastWarnTime[puzzle.Name] = time.Now().Add(h.cfg.InitialWarningDelay - h.cfg.MinWarningFrequency)
	} else if time.Since(lastWarning) <= h.cfg.MinWarningFrequency {
		return nil
	}
	var msgs []string
	if puzzle.PuzzleURL == "" {
		msgs = append(msgs, "missing a URL")
	}
	if puzzle.Round.Name == "" {
		msgs = append(msgs, "missing a round")
	}
	if len(msgs) == 0 {
		return fmt.Errorf("cannot warn about well-formatted puzzle %q: %v", puzzle.Name, puzzle)
	}
	if err := h.discord.QMChannelSend(fmt.Sprintf("Puzzle %q is %s", puzzle.Name, strings.Join(msgs, " and "))); err != nil {
		return err
	}
	h.lastWarnTime[puzzle.Name] = time.Now()
	return nil
}

func (h *HuntBot) markSolved(ctx context.Context, puzzle *schema.Puzzle) error {
	verb := "solved"
	if puzzle.Status == schema.Backsolved {
		verb = "backsolved"
	}

	if puzzle.Answer == "" {
		if err := h.discord.ChannelSend(puzzle.DiscordChannel, fmt.Sprintf("Puzzle %s!  Please add the answer to the sheet.", verb)); err != nil {
			return fmt.Errorf("error posting solved puzzle announcement: %v", err)
		}

		if err := h.discord.QMChannelSend(fmt.Sprintf("Puzzle %q marked %s, but has no answer, please add it to the sheet.", puzzle.Name, verb)); err != nil {
			return fmt.Errorf("error posting solved puzzle announcement: %v", err)
		}

		return nil // don't archive until we have the answer.
	}

	archived, err := h.discord.ArchiveChannel(puzzle.DiscordChannel)
	if !archived {
		// Channel already archived (cache is best-effort -- this can happen
		// after restart or if a human did it)
	} else if err != nil {
		return fmt.Errorf("unable to archive channel for %q: %v", puzzle.Name, err)
	} else {
		log.Printf("Archiving channel for %q", puzzle.Name)
		// post to relevant channels only if it was newly archived.
		if err := h.discord.ChannelSend(puzzle.DiscordChannel, fmt.Sprintf("Puzzle %s! The answer was `%v`. I'll archive this channel.", verb, puzzle.Answer)); err != nil {
			return fmt.Errorf("error posting solved puzzle announcement: %v", err)
		}

		embed := &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name:    fmt.Sprintf("Puzzle %s!", verb),
				IconURL: puzzle.Round.TwemojiURL(),
			},
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Channel",
					Value:  fmt.Sprintf("<#%s>", puzzle.DiscordChannel),
					Inline: true,
				},
				{
					Name:   "Answer",
					Value:  fmt.Sprintf("`%s`", puzzle.Answer),
					Inline: true,
				},
			},
		}

		if err := h.discord.GeneralChannelSendEmbed(embed); err != nil {
			return fmt.Errorf("error posting solved puzzle announcement: %v", err)
		}
	}

	log.Printf("Marking sheet solved for %q", puzzle.Name)
	err = h.drive.MarkSheetSolved(ctx, puzzle.SpreadsheetID)
	if err != nil {
		return err
	}

	h.archive(puzzle.Name)

	return nil
}

func (h *HuntBot) notifyNewPuzzle(puzzle *schema.Puzzle, channelID string) error {
	log.Printf("Posting information about new puzzle %q", puzzle.Name)

	// Pin a message with the spreadsheet URL to the channel
	if _, err := h.setPinnedStatusInfo(puzzle, channelID); err != nil {
		return fmt.Errorf("error pinning puzzle info: %v", err)
	}

	// Post a message in the general channel with a link to the puzzle.
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{
			Name:    "A new puzzle is available!",
			IconURL: puzzle.Round.TwemojiURL(),
		},
		Title: puzzle.Name,
		URL:   puzzle.PuzzleURL,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Channel",
				Value:  fmt.Sprintf("<#%s>", channelID),
				Inline: true,
			},
			{
				Name:   "Puzzle",
				Value:  fmt.Sprintf("[Link](%s)", puzzle.PuzzleURL),
				Inline: true,
			},
			{
				Name:   "Sheet",
				Value:  fmt.Sprintf("[Link](%s)", puzzle.SpreadsheetURL()),
				Inline: true,
			},
		},
	}
	if err := h.discord.GeneralChannelSendEmbed(embed); err != nil {
		return fmt.Errorf("error posting new puzzle announcement: %v", err)
	}

	return nil
}

func (h *HuntBot) isArchived(puzzleName string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.archived[puzzleName]
}

func (h *HuntBot) archive(puzzleName string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.archived[puzzleName] = true
}

func (h *HuntBot) setPuzzleStatus(name string, newStatus schema.Status) (oldStatus schema.Status) {
	h.mu.Lock()
	defer h.mu.Unlock()
	oldStatus = h.puzzleStatus[name]
	h.puzzleStatus[name] = newStatus
	return oldStatus
}

// logStatus marks the status; it is *not* called if the puzzle is solved
func (h *HuntBot) logStatus(ctx context.Context, puzzle *schema.Puzzle) error {
	didUpdate, err := h.setPinnedStatusInfo(puzzle, puzzle.DiscordChannel)
	if err != nil {
		return fmt.Errorf("unable to set puzzle status message for %q: %w", puzzle.Name, err)
	}

	if didUpdate {
		if err := h.discord.StatusUpdateChannelSend(fmt.Sprintf("%s Puzzle <#%s> is now %v.", puzzle.Round.Emoji, puzzle.DiscordChannel, puzzle.Status.Pretty())); err != nil {
			return fmt.Errorf("error posting puzzle status announcement: %v", err)
		}
	}

	return nil
}

const pinnedStatusHeader = "Puzzle Information"

func (h *HuntBot) setPinnedStatusInfo(puzzle *schema.Puzzle, channelID string) (didUpdate bool, err error) {
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: pinnedStatusHeader},
		Title:  puzzle.Name,
		URL:    puzzle.PuzzleURL,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Round",
				Value:  fmt.Sprintf("%v %v", puzzle.Round.Emoji, puzzle.Round.Name),
				Inline: false,
			},
			{
				Name:   "Status",
				Value:  puzzle.Status.Pretty(),
				Inline: true,
			},
			{
				Name:   "Puzzle",
				Value:  fmt.Sprintf("[Link](%s)", puzzle.PuzzleURL),
				Inline: true,
			},
			{
				Name:   "Sheet",
				Value:  fmt.Sprintf("[Link](%s)", puzzle.SpreadsheetURL()),
				Inline: true,
			},
		},
	}

	return h.discord.CreateUpdatePin(channelID, pinnedStatusHeader, embed)
}
