package huntbot

import (
	"github.com/gauravjsingh/emojihunt/discord"
	"github.com/gauravjsingh/emojihunt/drive"
)

type HuntBot struct {
	dis   *discord.Client
	drive *drive.Drive
}

func New(dis *discord.Client, drive *drive.Drive) *HuntBot {
	return &HuntBot{dis, drive}
}

func (h *HuntBot) CreatePuzzle() {
}

func (h *HuntBot) StartWork() {
	// register discord handlers to do work based on discord messages.
	//registerHandlers(h.dis)
	for _, handler := range discordHandlers {
		h.dis.RegisterNewMessageHandler(handler)
	}

	// poll the sheet and trigger work based on the polling.
	// Ideally, this would only look at changes, but we start with looking at everything.
}
