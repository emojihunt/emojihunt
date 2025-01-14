package status

import (
	"fmt"

	"golang.org/x/xerrors"
)

type Status string

const (
	NotStarted Status = ""
	Working    Status = "Working"
	Abandoned  Status = "Abandoned"
	Solved     Status = "Solved"
	Backsolved Status = "Backsolved"
	Purchased  Status = "Purchased"
)

const AlternateNotStarted = "Not Started"

func (s Status) IsValid() bool {
	switch s {
	case NotStarted, Working, Abandoned, Solved, Backsolved, Purchased:
		return true
	default:
		return false
	}
}

func ParseText(textPart string) (Status, error) {
	switch textPart {
	case string(NotStarted):
		return NotStarted, nil
	case string(AlternateNotStarted):
		return NotStarted, nil
	case string(Working):
		return Working, nil
	case string(Abandoned):
		return Abandoned, nil
	case string(Solved):
		return Solved, nil
	case string(Backsolved):
		return Backsolved, nil
	case string(Purchased):
		return Purchased, nil
	default:
		return NotStarted, xerrors.Errorf("unknown status %q", textPart)
	}
}

func (s Status) Pretty() string {
	switch s {
	case NotStarted:
		return "Not Started"
	default:
		return fmt.Sprintf("%s %s", s.Emoji(), s)
	}
}

func (s Status) Emoji() string {
	switch s {
	case NotStarted:
		return ""
	case Working:
		return "✍️"
	case Abandoned:
		return "🗑️"
	case Solved:
		return "🏅"
	case Backsolved:
		return "🤦‍♀️"
	case Purchased:
		return "💸"
	default:
		panic(xerrors.Errorf("called Emoji() on unknown status %q", s))
	}
}

func (s Status) IsSolved() bool {
	return s == Solved || s == Backsolved || s == Purchased
}

func (s Status) SolvedVerb() string {
	switch s {
	case Solved:
		return "solved!"
	case Backsolved:
		return "backsolved!!"
	case Purchased:
		return "purchased."
	default:
		panic("called SolvedVerb() on an unsolved puzzle")
	}
}

func (s Status) SolvedNoun() string {
	switch s {
	case Solved:
		return "solve!"
	case Backsolved:
		return "backsolve!!"
	case Purchased:
		return "free answer."
	default:
		panic("called SolvedExclamation() on an unsolved puzzle")
	}
}
