package huntbot

import (
	"github.com/gauravjsingh/emojihunt/discord/discordclient"
	"github.com/gauravjsingh/emojihunt/discord/drive"
)

type HuntBot struct {
	dis   *discordclient.DiscordClient
	drive *drive.Drive
}

func New(dis *discordclient.DiscordClient, drive *drive.Drive) *HuntBot {
	return &HuntBot{dis, drive}
}

func (h *HuntBot) CreatePuzzle() {
}

func (h *HuntBot) StartWork() {
	// register discord handlers to do work based on discord messages.

	// poll the sheet and trigger work based on the polling.
	// Ideally, this would only look at changes, but we start with looking at everything.
}
