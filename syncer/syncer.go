package syncer

import (
	"github.com/gauravjsingh/emojihunt/client"
)

type Syncer struct {
	airtable *client.Airtable
	discord  *client.Discord
	drive    *client.Drive
}

func New(airtable *client.Airtable, discord *client.Discord, drive *client.Drive) *Syncer {
	return &Syncer{airtable, discord, drive}
}
