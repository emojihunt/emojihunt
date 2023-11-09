package util

import (
	"time"
)

var BostonTime = MustLoadLocation("America/New_York")

type InvalidPuzzle struct {
	ID       int64
	Name     string
	Problems []string
	EditURL  string
}

func MustLoadLocation(name string) *time.Location {
	location, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic("could not load time zone: " + name)
	}
	return location
}
