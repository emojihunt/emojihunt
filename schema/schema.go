package schema

import (
	"fmt"
	"strings"

	"github.com/mehanizm/airtable"
)

type Puzzle struct {
	Name   string
	Answer string
	Round  Round
	Status Status

	AirtableRecord *airtable.Record
	PuzzleURL      string
	SpreadsheetID  string
	DiscordChannel string
	LastBotStatus  Status
}

func (p Puzzle) SpreadsheetURL() string {
	return fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", p.SpreadsheetID)
}

func (p Puzzle) IsValid() bool {
	return p.Name != "" && p.Round.Name != "" && p.PuzzleURL != ""
}

type Round struct {
	Name  string
	Emoji string
}

func ParseRound(raw string) Round {
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 {
		// Return an empty Round object; we have to check for this (see
		// IsValid(), above) and notify the QM so they can fix it.
		return Round{}
	}
	return Round{parts[1], parts[0]}
}

func (r Round) TwemojiURL() string {
	codePoints := make([]string, 0)
	for _, runeValue := range r.Emoji {
		codePoints = append(codePoints, fmt.Sprintf("%04x", runeValue))
	}
	return fmt.Sprintf("https://twemoji.maxcdn.com/2/72x72/%s.png", strings.Join(codePoints, "-"))
}

type Status int

const (
	NotStarted Status = iota
	Working
	Abandoned
	Solved
	Backsolved
	Archived
)

func ParseStatus(raw string) (Status, error) {
	if raw == "" {
		return NotStarted, nil
	}

	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 {
		err := fmt.Errorf("couldn't extract emoji and status from %#v", raw)
		panic(err)
	}

	switch parts[1] {
	case "Working":
		return Working, nil
	case "Abandoned":
		return Abandoned, nil
	case "Solved":
		return Solved, nil
	case "Backsolved":
		return Backsolved, nil
	case "Archived":
		return Archived, nil
	default:
		return NotStarted, fmt.Errorf("unknown status %v", raw)
	}
}

func (s Status) IsSolved() bool {
	return s == Solved || s == Backsolved
}

func (s Status) Pretty() string {
	switch s {
	case NotStarted:
		return "Not Started"
	default:
		return s.Serialize()
	}
}

func (s Status) Serialize() string {
	switch s {
	case NotStarted:
		return ""
	case Working:
		return "✍️ Working"
	case Abandoned:
		return "🗑️ Abandoned"
	case Solved:
		return "🏅 Solved"
	case Backsolved:
		return "🤦‍♀️ Backsolved"
	case Archived:
		return "📦 Archived"
	default:
		err := fmt.Errorf("unknown status %#v", s)
		panic(err)
	}
}
