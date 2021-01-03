package huntbot

import (
	"context"
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/gauravjsingh/emojihunt/discord"
	"github.com/gauravjsingh/emojihunt/drive"
)

type HuntBot struct {
	dis      *discord.Client
	drive    *drive.Drive
	handlers []func(*discordgo.Session, *discordgo.MessageCreate)
}

func New(dis *discord.Client, drive *drive.Drive) *HuntBot {
	return &HuntBot{dis: dis, drive: drive}
}

func (h *HuntBot) AddHandler(handler func(*discordgo.Session, *discordgo.MessageCreate)) {
	h.handlers = append(h.handlers, handler)
}

// TODO: is calling this after polling the sheet okay? every typo will turn into a sheet + channel
func (h *HuntBot) CreatePuzzle(ctx context.Context, name string) error {
	id, err := h.dis.CreateChannel(name)
	if err != nil {
		return fmt.Errorf("error creating discord channel for %q: %v", name, err)
	}
	// Create Spreadsheet
	sheetURL := "TODO: new puzzle URL"
	// If via bot, also take puzzle url as a param
	puzzleURL := ""
	// Update Spreadsheet with channel URL, spreadsheet URL.

	// Post a message in the general channel with a link to the puzzle.
	h.dis.GeneralChannelSend(fmt.Sprintf("There is a new puzzle %s! [Puzzle](%s), channel [#%s](%s)",
		name, puzzleURL, name, h.dis.ChannelURL(id)))
	// Pin a message with the spreadsheet URL to the channel
	h.dis.ChannelSendAndPin(id, fmt.Sprintf("[Spreadsheet](%s), [Puzzle](%s)", sheetURL, puzzleURL))
	return nil
}

func (h *HuntBot) StartWork(ctx context.Context) {
	// register discord handlers to do work based on discord messages.
	//registerHandlers(h.dis)
	for _, handler := range h.handlers {
		h.dis.RegisterNewMessageHandler(handler)
	}

	// poll the sheet and trigger work based on the polling.
	// Ideally, this would only look at changes, but we start with looking at everything.
	select {
	case <-ctx.Done():
	}
}
