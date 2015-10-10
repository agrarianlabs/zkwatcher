// Package zkwatcher provides an easy way to recursively watch
// a zookeeper path with meaningful events.
package zkwatcher

import (
	"fmt"
	"path"
	"sync"

	"github.com/samuel/go-zookeeper/zk"
)

// Watcher provides a read-only chan `C` where all the watched events are sent to.
type Watcher struct {
	C <-chan Event // Exposed read channel

	conn             *zk.Conn            // Zookeeper connection.
	lock             sync.Mutex          // Set lock.
	childrenWatchers map[string]struct{} // Set registering the watched childs.
	nodeWatchers     map[string]struct{} // Set registering the watched nodes.
	stopChan         chan struct{}       // Close control chan.
	ch               chan Event          // Internal channel for writes/close.
	wg               sync.WaitGroup      // Used to wait in Close for everything to be terminated.
	count            *int64              // Goroutine count for the watcher.
}

// NewWatcher initializes a watcher. It is the caller responsibility to Close it.
func NewWatcher(conn *zk.Conn) *Watcher {
	ch := make(chan Event, 1024) // arbitrary size, should be consumed and always be 0 len.
	return &Watcher{
		conn:             conn,
		childrenWatchers: map[string]struct{}{},
		nodeWatchers:     map[string]struct{}{},
		stopChan:         make(chan struct{}),
		C:                ch,
		ch:               ch,
		count:            new(int64),
	}
}

// WatchLimit creates a recursive watcher on the given zookeeper path.
// - `limit` indicates the maximum depth to go to. -1 for no limit.
// NOTE: require an existing `zkPath`.
func (wa *Watcher) WatchLimit(zkPath string, limit int) error {
	return wa.watch(zkPath, 0, limit)
}

// Watch creates a recursive watcher on the given zookeeper path.
// NOTE: require an existing `zkPath`.
// NOTE: there is no limit on the depth of the watcher, so the amount of gorountines
//       can be big.
func (wa *Watcher) Watch(zkPath string) error {
	return wa.WatchLimit(zkPath, -1)
}

// Close terminates the watcher.
// Blocks until every goroutines are gone.
func (wa *Watcher) Close() error {
	select {
	case <-wa.stopChan:
		return fmt.Errorf("already closed")
	default:
	}
	close(wa.stopChan) // Signal the termination to everyone.
	wa.wg.Wait()       // Wait for everyone to finish.
	close(wa.ch)       // Close the notify chan.
	return nil
}

// watchNode starts a watcher on the given node and all it's children.
// The first time, Created event gets reported.
// If the notify channel is full, the event gets discarded after 10 seconds.
//
// TODO: embed the data as part of the event?
//       would need to sent the first event *after* the GetW.
func (wa *Watcher) watchNode(zkPath string, lvl, limit int) {
	// Check if we don't already watch.
	wa.lock.Lock()
	if _, ok := wa.nodeWatchers[zkPath]; ok { // If we do, terminate the routine, decrement waitgroup and unlock.
		wa.wgDone()
		wa.lock.Unlock()
		return
	}
	wa.nodeWatchers[zkPath] = struct{}{} // Store the watched path in the set.
	wa.lock.Unlock()
	defer func() { // When the routine gets terminated, decrement the waigroup and remove self form the set.
		wa.wgDone()
		wa.lock.Lock()
		delete(wa.nodeWatchers, zkPath)
		wa.lock.Unlock()
	}()

	// First time, send a create event.
	wa.reportEvent(Event{Path: zkPath, Type: Create})

	for { // The watcher from ZK are single use, recreate one after each event.
		// Start a watcher.
		_, _, nodeEventCh, err := wa.conn.GetW(zkPath)
		if err != nil {
			wa.reportEvent(Event{Path: zkPath, Error: err})
			return // Error from zookeeper, interrupt the watcher.
		}

		// Start a watcher for the children of the node.
		if !wa.wgAdd() { // Increment the waitgroup.
			return // If the watcher is stopped, return.
		}
		go wa.watchChildren(zkPath, lvl, limit)

		// Wait for an event from ZK.
		var nodeEvent zk.Event
		select {
		case <-wa.stopChan: // If the watcher is closed, return.
			return
		case nodeEvent = <-nodeEventCh: // Got an event from ZK.
		}

		// Forge our own Event from the ZK one.
		event := Event{
			Path:  zkPath,
			Error: nodeEvent.Err,
		}
		// Map zk Event type to custom type.
		switch nodeEvent.Type {
		case zk.EventNodeCreated:
			event.Type = Create
		case zk.EventNodeDeleted:
			event.Type = Delete
		case zk.EventNodeDataChanged:
			event.Type = Update
		default:
			wa.reportEvent(Event{Path: zkPath, Error: fmt.Errorf("unknown zk event: %d", nodeEvent.Type)})
			continue // Unexpected behavior, do not interrupt the watcher.
		}

		// Send event to the user channel.
		wa.reportEvent(event)
	}
}

// watchChildren looks up the children for the given path,
// create an internal watcher on it and start a watcher on all children.
func (wa *Watcher) watchChildren(zkPath string, lvl, limit int) {
	if limit != -1 && lvl > limit { // If we are over the depth limit, stop here.
		wa.wgDone() // Decrement the waitgroup.
		return
	}
	// Check if we don't already watch.
	wa.lock.Lock()
	if _, ok := wa.childrenWatchers[zkPath]; ok { // If we do, terminate the routine, decrement waitgroup and unlock.
		wa.wgDone()
		wa.lock.Unlock()
		return
	}
	wa.childrenWatchers[zkPath] = struct{}{} // Store the watched path in the set.
	wa.lock.Unlock()
	defer func() { // When the routine gets terminated, decrement the waigroup and remove self form the set.
		wa.wgDone()
		wa.lock.Lock()
		delete(wa.childrenWatchers, zkPath)
		wa.lock.Unlock()
	}()

	for { // The watcher from ZK are single use, recreate one after each event.
		// Start a watcher.
		children, _, childEventCh, err := wa.conn.ChildrenW(zkPath)
		if err != nil {
			wa.reportEvent(Event{Path: zkPath, Error: err})
			return // Error from zookeeper, interrupt the watcher.
		}
		// Start a node watcher for each of the children.
		for _, child := range children {
			if !wa.wgAdd() { // Increment the waitgroup.
				return // If the watcher is stopped, return.
			}
			go wa.watchNode(path.Join(zkPath, child), lvl+1, limit)
		}

		select {
		case <-wa.stopChan: // If the watcher is stopped, return.
			return
		case childEvent := <-childEventCh: // Get an event from ZK.
			// If the node gets deleted, return.
			if childEvent.Type == zk.EventNodeDeleted {
				return
			}

			// Discard the event.
			// This will restart a watcher for the children and
			// start the node watcher for any new children.
			// Terminated child gets automatically removed.
		}

	}
}

// watch creates a watcher on the given zookeeper path.
// NOTE: require an existing `zkPath`.
func (wa *Watcher) watch(zkPath string, lvl, limit int) error {
	// Make sure we are not already closed.
	select {
	case <-wa.stopChan:
		return fmt.Errorf("watcher closed")
	default:
	}
	// Sanitize the path.
	if len(zkPath) > 0 && zkPath[len(zkPath)-1] == '/' {
		zkPath = zkPath[:len(zkPath)-1]
	}
	// Make sure the path exists.
	if exist, _, err := wa.conn.Exists(zkPath); err != nil {
		return fmt.Errorf("error looking up watch path %q: %s", zkPath, err)
	} else if !exist {
		return fmt.Errorf("path %q does not exist", zkPath)
	}
	// Start the watcher.
	if !wa.wgAdd() { // Increment the waitgroup.
		return nil // If the watcher is stopped, return.
	}
	go wa.watchNode(zkPath, lvl, limit)
	return nil
}
