package ki

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Debouncer to handle event batching
type debouncer struct {
	mu       sync.Mutex
	timer    *time.Timer
	events   []fsnotify.Event
	duration time.Duration
	callback func(events []fsnotify.Event)
}

func new_debouncer(duration time.Duration, callback func(events []fsnotify.Event)) *debouncer {
	return &debouncer{duration: duration, callback: callback}
}

func (d *debouncer) add_evt(event fsnotify.Event) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.events = append(d.events, event)

	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.duration, func() {
		d.mu.Lock()
		defer d.mu.Unlock()

		if len(d.events) > 0 {
			d.callback(d.events)
			d.events = nil
		}
	})
}
