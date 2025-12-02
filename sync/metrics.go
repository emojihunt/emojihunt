package sync

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	puzzleQueueLen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sync_puzzle_queue",
		Help: "The length of the puzzle-sync queue",
	})
	roundQueueLen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sync_round_queue",
		Help: "The length of the round-sync queue",
	})
	liveQueueLen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "sync_live_queue",
		Help: "The length of the live-message queue",
	})

	discoveryRestarts = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sync_discovery_count",
		Help: "The total number of times discovery has been restarted",
	})
	puzzlesProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sync_puzzle_count",
		Help: "The total number of puzzles synced",
	})
	roundsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "sync_round_count",
		Help: "The total number of puzzles synced",
	})
)

func (c *Client) HandleMetrics() {
	for {
		puzzleQueueLen.Set(float64(len(c.state.PuzzleChange)))
		roundQueueLen.Set(float64(len(c.state.RoundChange)))
		liveQueueLen.Set(float64(len(c.state.LiveMessage)))
		time.Sleep(1 * time.Second)
	}
}
