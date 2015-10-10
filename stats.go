package zkwatcher

import "sync/atomic"

// Stats contains all runtime data from the watcher.
type Stats struct {
	Depth           int      `json:"depth"`
	Cap             int      `json:"cap"`
	Goroutines      int      `json:"goroutines"`
	Running         bool     `json:"running"`
	WatchedNodes    []string `json:"watched_nodes"`
	WatchedChildren []string `json:"watched_children"`
}

// Stats .
func (wa *Watcher) Stats() Stats {
	s := Stats{}

	select {
	case <-wa.stopChan:
		return s
	default:
	}

	s.WatchedNodes = make([]string, 0, len(wa.nodeWatchers))
	for node := range wa.nodeWatchers {
		s.WatchedNodes = append(s.WatchedNodes, node)
	}
	s.WatchedChildren = make([]string, 0, len(wa.childrenWatchers))
	for child := range wa.childrenWatchers {
		s.WatchedChildren = append(s.WatchedChildren, child)
	}
	s.Running = true
	s.Goroutines = int(atomic.LoadInt64(wa.count))
	s.Depth = len(wa.ch)
	s.Cap = cap(wa.ch)

	return s
}
