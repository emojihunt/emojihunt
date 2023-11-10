package db

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

func (p Puzzle) Title() string {
	// Puzzle name for Discord channel, spreadsheet, etc. (may be an abbreviated
	// version of the full name, specified by the QM)
	if p.NameOverride != "" {
		return p.NameOverride
	}
	return p.Name
}

func (p Puzzle) RoundName() string {
	return "TODO: Round Name"
}

func (p Puzzle) RoundEmoji() string {
	return "🐞" // TODO
}

func (p Puzzle) SpreadsheetURL() string {
	if p.SpreadsheetID == "" {
		panic("called SpreadsheetURL() on a puzzle with no spreadsheet")
	}
	return fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", p.SpreadsheetID)
}

func (p Puzzle) EditURL() string {
	return "https://todo.emojihunt.tech/TODO"
}

func (p Puzzle) ShouldArchive() bool {
	// We shouldn't archive the channel until the answer has been filled in on
	// Airtable
	return p.Status.IsSolved() && p.Answer != ""
}

var categories = []string{"A", "B", "C"}

func (p Puzzle) ArchiveCategory() string {
	// Hash the Discord channel ID, since it's not totally random
	h := sha256.New()
	if _, err := h.Write([]byte(p.DiscordChannel)); err != nil {
		panic(err)
	}
	i := binary.BigEndian.Uint64(h.Sum(nil)[:8])

	return categories[i%uint64(len(categories))]
}
