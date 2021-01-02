package huntbot

import (
	"context"

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
