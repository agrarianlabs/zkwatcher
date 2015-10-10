// +build integration

package zkwatcher

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

// TestWatcherCloseDirect starts a watcher and close it directly.
// This reuslt in terminatting while still initializing and should work fine anyway.
func TestWatcherCloseDirect(t *testing.T) {
	conn, _, err := zk.Connect([]string{testZKHost}, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	root := "/" + randomName()
	createTree(t, conn, root+"/testwatcher/sub/tree")
	defer removeTree(t, conn, root)

	watcher := NewWatcher(conn)

	if err := watcher.WatchLimit(root+"/testwatcher", 2); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt64(watcher.count) == 0 {
		t.Fatal("goroutines count should not be 0")
	}
	if err := watcher.Close(); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt64(watcher.count) != 0 {
		t.Fatalf("non 0 goroutines count after close: %d", *watcher.count)
	}
}

// TestWatcherCloseTimed starts a watcher, wait for 2 seconds to make sure
// all routines are started and close.
func TestWatcherCloseTimed(t *testing.T) {
	if testing.Short() {
		t.Skip("short testing, skipping timed closed test")
	}

	conn, _, err := zk.Connect([]string{testZKHost}, 10*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	root := "/" + randomName()
	createTree(t, conn, root+"/testwatcher/sub/tree")
	defer removeTree(t, conn, root)

	watcher := NewWatcher(conn)

	if err := watcher.WatchLimit(root+"/testwatcher/sub/tree", 2); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	if atomic.LoadInt64(watcher.count) == 0 {
		t.Fatal("goroutines count should not be 0")
	}
	if err := watcher.Close(); err != nil {
		t.Fatal(err)
	}
	if atomic.LoadInt64(watcher.count) != 0 {
		t.Fatalf("non 0 goroutines count after close: %d", *watcher.count)
	}
}
