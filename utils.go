package zkwatcher

import (
	"sync/atomic"
	"time"
)

// wgAdd adds 1 to the waitgroup if the watcher is not closed.
// returns true if the watcher up and running.
// Also keep track of the amount of goroutines running.
func (wa *Watcher) wgAdd() bool {
	select {
	case <-wa.stopChan:
		return false
	default:
		wa.wg.Add(1)
		atomic.AddInt64(wa.count, 1)
		return true
	}
}

// wgDone remove one from the waitground and the goroutine count.
func (wa *Watcher) wgDone() {
	atomic.AddInt64(wa.count, -1)
	wa.wg.Done()
}

// reportEvent sends the given event to the user channel.
// If the channel is full, event gets discarded after 10 seconds.
func (wa *Watcher) reportEvent(event Event) {
	// Send the event to the channel in a short lived goroutine.
	if !wa.wgAdd() {
		return
	}
	go func() {
		// Send the event with a 10 seconds timeout.
		timeout := time.NewTicker(10 * time.Second)
		select { // It can unblock when: 1. watcher closed, 2. timeout, 3. event sent.
		case <-wa.stopChan:
		case <-timeout.C:
		case wa.ch <- event:
		}
		timeout.Stop()
		wa.wgDone()
	}()
}
