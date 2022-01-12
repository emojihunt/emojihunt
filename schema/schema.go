package schema

import (
	"fmt"
	"strings"
	"time"

	"github.com/mehanizm/airtable"
)

type Puzzle struct {
	Name         string
	Answer       string
	Rounds       Rounds
	Status       Status
	Description  string
	Notes        string
	NameOverride string

	AirtableRecord *airtable.Record
	PuzzleURL      string
	SpreadsheetID  string
	DiscordChannel string

	Pending       bool
	LastBotStatus Status
	Archived      bool
	LastBotSync   *time.Time

	OriginalURL string
	VoiceRoom   string

	LastModified   *time.Time
	LastModifiedBy string // user id

	Unlock func()
}

type NewPuzzle struct {
	Name      string
	Round     Round
	PuzzleURL string
}

type VoicePuzzle struct {
	RecordID  string
	Name      string
	VoiceRoom string
}

func (p Puzzle) SpreadsheetURL() string {
	if p.SpreadsheetID == "" {
		panic("called SpreadsheetURL() on a puzzle with no spreadsheet")
	}
	return fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", p.SpreadsheetID)
}

func (p Puzzle) IsValid() bool {
	return p.Name != "" && len(p.Rounds) > 0 &&
		p.PuzzleURL != "" && (p.Status.IsSolved() || p.Answer == "")
}

func (p Puzzle) ShouldArchive() bool {
	// We shouldn't archive the channel until the answer has been filled in on
	// Airtable
	return p.Status.IsSolved() && p.Answer != ""
}

func (p Puzzle) Title() string {
	// Puzzle name for Discord channel, spreadsheet, etc. (may be an abbreviated
	// version of the full name, specified by the QM)
	if p.NameOverride != "" {
		return p.NameOverride
	}
	return p.Name
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

type Rounds []Round

func (rs Rounds) Len() int           { return len(rs) }
func (rs Rounds) Less(i, j int) bool { return rs[i].Name < rs[j].Name }
func (rs Rounds) Swap(i, j int)      { rs[i], rs[j] = rs[j], rs[i] }

func (rs Rounds) Emojis() string {
	var emojis []string
	for _, r := range rs {
		emojis = append(emojis, r.Emoji)
	}
	return strings.Join(emojis, "")
}

func (rs Rounds) Names() string {
	var names []string
	for _, r := range rs {
		names = append(names, r.Name)
	}
	return strings.Join(names, "–")
}

func (rs Rounds) EmojisAndNames() []string {
	var result []string
	for _, r := range rs {
		result = append(result, fmt.Sprintf("%s %s", r.Emoji, r.Name))
	}
	return result
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
	case "Not Started":
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

func (s Status) Human() string {
	switch s {
	case NotStarted:
		return "Not Started"
	case "Working":
		return "✍️ Working"
	case "Abandoned":
		return "🗑️ Abandoned"
	case "Solved":
		return "🏅 Solved"
	case "Backsolved":
		return "🤦‍♀️ Backsolved"
	default:
		panic(fmt.Errorf("called Human() on unknown status %v", s))
	}
}

func (s Status) PrettyForAirtable() interface{} {
	if s == NotStarted {
		return nil
	}
	return s.Human()
}

func (s Status) TextForAirtable() interface{} {
	if s == NotStarted {
		return nil
	}
	return string(s)
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

func (s Status) SolvedNoun() string {
	switch s {
	case Solved:
		return "solve"
	case Backsolved:
		return "backsolve"
	default:
		panic("called SolvedNoun() on an unsolved puzzle")
	}
}
