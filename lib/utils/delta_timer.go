package utils

import "time"

type DeltaTimer struct {
	time.Time
}

func (d *DeltaTimer) Next() time.Duration {
	// acquire timestamp exactly once to ensure we're not accumulating error
	now := time.Now()

	defer d.Set(now)
	if d.IsZero() {
		return 0
	}
	return now.Sub(d.Time)
}

func (d *DeltaTimer) Set(t time.Time) {
	d.Time = t
}
