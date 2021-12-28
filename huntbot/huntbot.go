package huntbot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/airtable"
	"github.com/gauravjsingh/emojihunt/discord"
	"github.com/gauravjsingh/emojihunt/drive"
	"github.com/gauravjsingh/emojihunt/schema"
)

type Config struct {
	// How often to warn in discord about badly formatted puzzles.
	MinWarningFrequency time.Duration
	InitialWarningDelay time.Duration
	UpdateRooms         bool
}

type HuntBot struct {
	dis      *discord.Client
	drive    *drive.Drive
	airtable *airtable.Client
	cfg      Config

	mu           sync.Mutex               // hold while accessing everything below
	enabled      bool                     // global killswitch, toggle with !huntbot kill/!huntbot start
	puzzleStatus map[string]schema.Status // name -> status (best-effort cache)
	archived     map[string]bool          // name -> channel was archived (best-effort cache)
	// When we last warned about a malformed puzzle.
	lastWarnTime map[string]time.Time
}

func New(dis *discord.Client, d *drive.Drive, airtable *airtable.Client, c Config) *HuntBot {
	return &HuntBot{
		dis:          dis,
		drive:        d,
		airtable:     airtable,
		enabled:      true,
		puzzleStatus: map[string]schema.Status{},
		archived:     map[string]bool{},
		lastWarnTime: map[string]time.Time{},
		cfg:          c,
	}
}

const pinnedStatusHeader = "Puzzle Information"

func (h *HuntBot) setPinnedStatusInfo(puzzle *schema.Puzzle, channelID string) (didUpdate bool, err error) {
	embed := &discordgo.MessageEmbed{
		Author: &discordgo.MessageEmbedAuthor{Name: pinnedStatusHeader},
		Color:  -1, // TODO
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

	return h.dis.CreateUpdatePin(channelID, pinnedStatusHeader, embed)
}

const roomStatusHeader = "Working Room"

func (h *HuntBot) setPinnedVoiceInfo(puzzleChannelID string, voiceChannelID *string) (didUpdate bool, err error) {
	room := "No voice room set. \"!room start $room\" to start working in $room."
	if voiceChannelID != nil {
		room = fmt.Sprintf("Join us in <#%s>!", *voiceChannelID)
	}
	embed := &discordgo.MessageEmbed{
		Author:      &discordgo.MessageEmbedAuthor{Name: roomStatusHeader},
		Description: room,
	}

	return h.dis.CreateUpdatePin(puzzleChannelID, roomStatusHeader, embed)
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
		Color: -1, // TODO
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
	if err := h.dis.GeneralChannelSendEmbed(embed); err != nil {
		return fmt.Errorf("error posting new puzzle announcement: %v", err)
	}

	return nil
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
		if err := h.dis.StatusUpdateChannelSend(fmt.Sprintf("%s Puzzle <#%s> is now %v.", puzzle.Round.Emoji, puzzle.DiscordChannel, puzzle.Status.Pretty())); err != nil {
			return fmt.Errorf("error posting puzzle status announcement: %v", err)
		}
	}

	return nil
}

func (h *HuntBot) markSolved(ctx context.Context, puzzle *schema.Puzzle) error {
	verb := "solved"
	if puzzle.Status == schema.Backsolved {
		verb = "backsolved"
	}

	if puzzle.Answer == "" {
		if err := h.dis.ChannelSend(puzzle.DiscordChannel, fmt.Sprintf("Puzzle %s!  Please add the answer to the sheet.", verb)); err != nil {
			return fmt.Errorf("error posting solved puzzle announcement: %v", err)
		}

		if err := h.dis.QMChannelSend(fmt.Sprintf("Puzzle %q marked %s, but has no answer, please add it to the sheet.", puzzle.Name, verb)); err != nil {
			return fmt.Errorf("error posting solved puzzle announcement: %v", err)
		}

		return nil // don't archive until we have the answer.
	}

	archived, err := h.dis.ArchiveChannel(puzzle.DiscordChannel)
	if !archived {
		// Channel already archived (cache is best-effort -- this can happen
		// after restart or if a human did it)
	} else if err != nil {
		return fmt.Errorf("unable to archive channel for %q: %v", puzzle.Name, err)
	} else {
		log.Printf("Archiving channel for %q", puzzle.Name)
		// post to relevant channels only if it was newly archived.
		if err := h.dis.ChannelSend(puzzle.DiscordChannel, fmt.Sprintf("Puzzle %s! The answer was `%v`. I'll archive this channel.", verb, puzzle.Answer)); err != nil {
			return fmt.Errorf("error posting solved puzzle announcement: %v", err)
		}

		embed := &discordgo.MessageEmbed{
			Author: &discordgo.MessageEmbedAuthor{
				Name:    fmt.Sprintf("Puzzle %s!", verb),
				IconURL: puzzle.Round.TwemojiURL(),
			},
			Color: -1, // TODO
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

		if err := h.dis.GeneralChannelSendEmbed(embed); err != nil {
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
	if err := h.dis.QMChannelSend(fmt.Sprintf("Puzzle %q is %s", puzzle.Name, strings.Join(msgs, " and "))); err != nil {
		return err
	}
	h.lastWarnTime[puzzle.Name] = time.Now()
	return nil
}

func (h *HuntBot) updatePuzzle(ctx context.Context, puzzle *schema.Puzzle) error {
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
		channel, err := h.dis.CreateChannel(puzzle.Name)
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
