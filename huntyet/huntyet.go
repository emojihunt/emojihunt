package huntyet

import (
	"time"

	"github.com/emojihunt/emojihunt/util"
)

const duration = 72 * time.Hour

var startTimes = []time.Time{
	// must be ordered oldest to newest!
	time.Date(2021, 1, 15, 12, 0, 0, 0, util.BostonTime),
	time.Date(2022, 1, 14, 12, 0, 0, 0, util.BostonTime),
	time.Date(2023, 1, 13, 12, 0, 0, 0, util.BostonTime),
	time.Date(2023, 1, 13, 12, 0, 0, 0, util.BostonTime),
	time.Date(2024, 1, 12, 12, 0, 0, 0, util.BostonTime),
}

// Returns the start time of the next Hunt, or nil if Hunt is ongoing. ok
// indicates whether the list of Hunts is current.
func NextHunt(at time.Time) (next *time.Time, ok bool) {
	for _, start := range startTimes {
		end := start.Add(duration)
		if at.Before(start) {
			return &start, true
		} else if at.Before(end) {
			return nil, true
		}
		// else: this hunt has passed, check the next hunt
	}
	return nil, false
}
