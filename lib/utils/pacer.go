package utils

import "time"

type Pacer struct {
	interval time.Duration
	nextCall time.Time
}

func NewPacer(interval time.Duration) *Pacer {
	return &Pacer{
		interval: interval,
	}
}

func (p *Pacer) Sleep() {
	now := time.Now()

	if now.Before(p.nextCall) {
		time.Sleep(p.nextCall.Sub(now))
	}

	p.nextCall = p.nextCall.Add(p.interval)

	// If we've fallen too far behind, reset to now + interval
	if p.nextCall.Before(time.Now()) {
		p.nextCall = time.Now().Add(p.interval)
	}
}
