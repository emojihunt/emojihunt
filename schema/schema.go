package schema

import (
	"fmt"
	"strings"
)

type Puzzle struct {
	Name   string
	Answer string
	Round  Round
	Status Status

	AirtableID     string
	PuzzleURL      string
	SpreadsheetID  string
	DiscordChannel string
}

func (p Puzzle) SpreadsheetURL() string {
	return fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", p.SpreadsheetID)
}

type Round struct {
	Name  string
	Emoji string
}

func ParseRound(raw string) Round {
	parts := strings.SplitN(raw, " ", 2)
	if len(parts) != 2 {
		err := fmt.Errorf("couldn't extract emoji and round name from %#v", raw)
		panic(err)
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

// func (r *Round) IntColor() int {
// 	red := int(r.Color.Red * 255)
// 	green := int(r.Color.Green * 255)
// 	blue := int(r.Color.Blue * 255)
// 	return (red << 16) + (green << 8) + blue
// }

// TODO: how should we support extending Status?
type Status string

const (
	Working    Status = "🏅Solved" // TODO: wait what?
	Abandoned  Status = "🗑️Abandoned"
	Solved     Status = "🏅Solved"
	Backsolved Status = "🤦‍♀️Backsolved"
)

func ParseStatus(s string) Status {
	if s == string(Working) {
		return Working
	} else if s == string(Abandoned) {
		return Abandoned
	} else if s == string(Solved) {
		return Solved
	} else if s == string(Backsolved) {
		return Backsolved
	} else {
		err := fmt.Errorf("unknown status %v", s)
		panic(err)
	}
}

func (s Status) IsSolved() bool {
	return s == Solved || s == Backsolved
}

func (s Status) Pretty() string {
	if string(s) == "" {
		return "Not Started"
	} else {
		return string(s)
	}
}
