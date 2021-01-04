package huntbot

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/discord"
	"github.com/gauravjsingh/emojihunt/drive"
)

type HuntBot struct {
	dis      *discord.Client
	drive    *drive.Drive
	handlers map[string]discord.NewMessageHandler
}

func New(dis *discord.Client, drive *drive.Drive) *HuntBot {
	return &HuntBot{dis: dis, drive: drive, handlers: make(map[string]discord.NewMessageHandler)}
}

func (h *HuntBot) AddHandler(name string, handler discord.NewMessageHandler) {
	h.handlers[name] = handler
}

// TODO: is calling this after polling the sheet okay? every typo will turn into a sheet + channel
func (h *HuntBot) NewPuzzle(ctx context.Context, name string) error {
	id, err := h.dis.CreateChannel(name)
	if err != nil {
		return fmt.Errorf("error creating discord channel for %q: %v", name, err)
	}
	// Create Spreadsheet
	sheetURL := "https://docs.google.com/spreadsheets/d/1SgvhTBeVdyTMrCR0wZixO3O0lErh4vqX0--nBpSfYT8/edit"
	// If via bot, also take puzzle url as a param
	puzzleURL := "https://en.wikipedia.org/wiki/Main_Page"
	// Update Spreadsheet with channel URL, spreadsheet URL.

	// Post a message in the general channel with a link to the puzzle.
	if err := h.dis.GeneralChannelSend(fmt.Sprintf("There is a new puzzle %s!\nPuzzle URL: %s\nChannel <#%s>", name, puzzleURL, id)); err != nil {
		return fmt.Errorf("error posting new puzzle announcement: %v", err)
	}
	// Pin a message with the spreadsheet URL to the channel
	if err := h.dis.ChannelSendAndPin(id, fmt.Sprintf("Spreadsheet: %s\nPuzzle: %s", sheetURL, puzzleURL)); err != nil {
		return fmt.Errorf("error pinning puzzle info: %v", err)
	}
	return nil
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

func (h *HuntBot) PollSheet(ctx context.Context) {
	// TODO
}
