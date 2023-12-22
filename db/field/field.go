package field

import "golang.org/x/xerrors"

type Status string

const (
	StatusNotStarted Status = ""
	StatusWorking    Status = "Working"
	StatusAbandoned  Status = "Abandoned"
	StatusSolved     Status = "Solved"
	StatusBacksolved Status = "Backsolved"
	StatusPurchased  Status = "Purchased"
)

const AlternateNotStarted = "Not Started"

func (s Status) IsValid() bool {
	switch s {
	case StatusNotStarted, StatusWorking, StatusAbandoned,
		StatusSolved, StatusBacksolved, StatusPurchased:
		return true
	default:
		return false
	}
}

func ParseTextStatus(textPart string) (Status, error) {
	switch textPart {
	case string(StatusNotStarted):
		return StatusNotStarted, nil
	case string(AlternateNotStarted):
		return StatusNotStarted, nil
	case string(StatusWorking):
		return StatusWorking, nil
	case string(StatusAbandoned):
		return StatusAbandoned, nil
	case string(StatusSolved):
		return StatusSolved, nil
	case string(StatusBacksolved):
		return StatusBacksolved, nil
	case string(StatusPurchased):
		return StatusPurchased, nil
	default:
		return StatusNotStarted, xerrors.Errorf("unknown status %q", textPart)
	}
}

func (s Status) Pretty() string {
	switch s {
	case StatusNotStarted:
		return "Not Started"
	case StatusWorking:
		return "✍️ Working"
	case StatusAbandoned:
		return "🗑️ Abandoned"
	case StatusSolved:
		return "🏅 Solved"
	case StatusBacksolved:
		return "🤦‍♀️ Backsolved"
	case StatusPurchased:
		return "💸 Purchased"
	default:
		panic(xerrors.Errorf("called Human() on unknown status %q", s))
	}
}

func (s Status) IsSolved() bool {
	return s == StatusSolved || s == StatusBacksolved || s == StatusPurchased
}

func (s Status) SolvedVerb() string {
	switch s {
	case StatusSolved:
		return "solved"
	case StatusBacksolved:
		return "backsolved"
	case StatusPurchased:
		return "purchased"
	default:
		panic("called SolvedVerb() on an unsolved puzzle")
	}
}

func (s Status) SolvedExclamation() string {
	switch s {
	case StatusSolved:
		return "solve!"
	case StatusBacksolved:
		return "backsolve!!"
	case StatusPurchased:
		return "free answer."
	default:
		panic("called SolvedExclamation() on an unsolved puzzle")
	}
}
