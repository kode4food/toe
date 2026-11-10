// Package debounce coalesces repeated calls per key into the last one
package debounce

import (
	"sync"
	"time"
)

// Keyed runs the last function given for a key once that key has gone quiet for the delay
type Keyed[K comparable] struct {
	mu      sync.Mutex
	delay   time.Duration
	timers  map[K]*time.Timer
	stopped bool
}

// NewKeyed creates a Keyed that waits delay after the last call per key
func NewKeyed[K comparable](delay time.Duration) *Keyed[K] {
	return &Keyed[K]{
		delay:  delay,
		timers: map[K]*time.Timer{},
	}
}

// Do schedules fn for key, replacing any call for key still waiting
func (k *Keyed[K]) Do(key K, fn func()) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.stopped {
		return
	}
	if t, ok := k.timers[key]; ok {
		t.Stop()
	}
	var t *time.Timer
	t = time.AfterFunc(k.delay, func() {
		k.mu.Lock()
		current := k.timers[key] == t
		if current {
			delete(k.timers, key)
		}
		k.mu.Unlock()
		if current {
			fn()
		}
	})
	k.timers[key] = t
}

// Cancel drops the call waiting for key
func (k *Keyed[K]) Cancel(key K) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if t, ok := k.timers[key]; ok {
		t.Stop()
		delete(k.timers, key)
	}
}

// Stop drops every waiting call and ignores later ones
func (k *Keyed[K]) Stop() {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.stopped = true
	for _, t := range k.timers {
		t.Stop()
	}
	clear(k.timers)
}
