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

	Pending       bool
	LastBotStatus Status
	Archived      bool
	OriginalURL   string
}

type NewPuzzle struct {
	Name      string
	Round     Round
	PuzzleURL string
}

func (p Puzzle) SpreadsheetURL() string {
	if p.SpreadsheetID == "" {
		panic("called SpreadsheetURL() on a puzzle with no spreadsheet")
	}
	return fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", p.SpreadsheetID)
}

func (p Puzzle) IsValid() bool {
	return p.Name != "" && p.Round.Name != "" &&
		p.PuzzleURL != "" && (p.Status.IsSolved() || p.Answer == "")
}

func (p Puzzle) ShouldArchive() bool {
	// We shouldn't archive the channel until the answer has been filled in on
	// Airtable
	return p.Status.IsSolved() && p.Answer != ""
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

func (r Round) Serialize() string {
	return r.Emoji + " " + r.Name
}

type Status string

const (
	NotStarted Status = ""
	Working    Status = "Working"
	Abandoned  Status = "Abandoned"
	Solved     Status = "Solved"
	Backsolved Status = "Backsolved"
)

func ParsePrettyStatus(raw string) (Status, error) {
	if raw == "" {
		return NotStarted, nil
	}

	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 {
		err := fmt.Errorf("couldn't extract emoji and status from %#v", raw)
		return NotStarted, err
	}
	return ParseTextStatus(parts[1])
}

func ParseTextStatus(textPart string) (Status, error) {
	switch textPart {
	case "":
		return NotStarted, nil
	case "Working":
		return Working, nil
	case "Abandoned":
		return Abandoned, nil
	case "Solved":
		return Solved, nil
	case "Backsolved":
		return Backsolved, nil
	default:
		return NotStarted, fmt.Errorf("unknown status %q", textPart)
	}
}

func (s Status) IsSolved() bool {
	return s == Solved || s == Backsolved
}

func (s Status) SolvedVerb() string {
	switch s {
	case Solved:
		return "solved"
	case Backsolved:
		return "backsolved"
	default:
		panic("called SolvedVerb() on an unsolved puzzle")
	}
}

func (s Status) Pretty() string {
	switch s {
	case NotStarted:
		return "Not Started"
	default:
		return string(s)
	}
}
