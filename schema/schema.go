package schema

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"golang.org/x/xerrors"
)

var BostonTime = MustLoadLocation("America/New_York")

type Puzzle struct {
	ID int64

	Name         string
	Answer       string
	Round        Round
	Status       Status
	Description  string
	Location     string
	NameOverride string

	PuzzleURL      string
	SpreadsheetID  string
	DiscordChannel string

	Archived bool

	OriginalURL string
	VoiceRoom   string
	Reminder    *time.Time
}

type NewPuzzle struct {
	Name        string
	Round       Round
	PuzzleURL   string
	OriginalURL string
}

type InvalidPuzzle struct {
	ID       int64
	Name     string
	Problems []string
	EditURL  string
}

type VoicePuzzle struct {
	ID        int64
	Name      string
	VoiceRoom string
}

type ReminderPuzzle struct {
	ID             int64
	Name           string
	DiscordChannel string
	Reminder       time.Time
}

type ReminderPuzzles []ReminderPuzzle

func (rps ReminderPuzzles) Len() int           { return len(rps) }
func (rps ReminderPuzzles) Less(i, j int) bool { return rps[i].Reminder.Before(rps[j].Reminder) }
func (rps ReminderPuzzles) Swap(i, j int)      { rps[i], rps[j] = rps[j], rps[i] }

func (p Puzzle) SpreadsheetURL() string {
	if p.SpreadsheetID == "" {
		panic("called SpreadsheetURL() on a puzzle with no spreadsheet")
	}
	return fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s", p.SpreadsheetID)
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

func (p Puzzle) Title() string {
	// Puzzle name for Discord channel, spreadsheet, etc. (may be an abbreviated
	// version of the full name, specified by the QM)
	if p.NameOverride != "" {
		return p.NameOverride
	}
	return p.Name
}

func (p Puzzle) Problems() []string {
	// TODO: make these mandatory validations
	var problems []string
	if p.Name == "" {
		problems = append(problems, "missing puzzle name")
	}
	if p.Round.Name == "" || p.Round.Emoji == "" {
		problems = append(problems, "invalid round (did you put a space between the "+
			"emoji and the round name?)")
	}
	if p.PuzzleURL == "" {
		problems = append(problems, "missing puzzle URL")
	}
	if p.Answer != "" && !p.Status.IsSolved() {
		problems = append(problems, "has an answer even though it's not solved")
	}
	return problems
}

type Round struct {
	Name  string
	Emoji string
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
		return NotStarted, xerrors.Errorf("unknown status %q", textPart)
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
		panic(xerrors.Errorf("called Human() on unknown status %q", s))
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

func MustLoadLocation(name string) *time.Location {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic("could not load time zone: " + name)
	}
	return location
}
