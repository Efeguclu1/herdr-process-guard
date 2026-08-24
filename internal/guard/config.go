package guard

import "time"

type Config struct {
	RecentStartAfter time.Duration
	IdleAfter        time.Duration
	OrphanAfter      time.Duration
	HistoryRetention time.Duration
	MaxObservations  int
	TerminateGrace   time.Duration
}

func DefaultConfig() Config {
	return Config{
		RecentStartAfter: 5 * time.Minute,
		IdleAfter:        10 * time.Minute,
		OrphanAfter:      30 * time.Minute,
		HistoryRetention: 30 * 24 * time.Hour,
		MaxObservations:  10_000,
		TerminateGrace:   5 * time.Second,
	}
}
