// +build integration

package zkwatcher

import (
	"flag"
	"math/rand"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

var testZKHost string

func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	// Get the testing zookeeper host from command line.
	flag.StringVar(&testZKHost, "zk-host", "comma separated testing zookeeper hosts", "<none>")

}

func randomName() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"

	result := make([]byte, 16)
	for i := range result {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return "zkwatchtest_" + string(result)
}

// createTree recursively creates the given path.
func createTree(t *testing.T, conn *zk.Conn, zkPath string) {
	target := "/"
	for _, elem := range strings.Split(zkPath, "/") {
		if elem != "" {
			target = path.Join(target, elem)
			if _, err := conn.Create(target, nil, 0, zk.WorldACL((zk.PermAll))); err != nil {
				t.Fatalf("error creating %q: %s", target, err)
			}
		}
	}
}

// removeTree recursively removes the given path.
func removeTree(t *testing.T, conn *zk.Conn, zkPath string) {
	children, _, err := conn.Children(zkPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, child := range children {
		removeTree(t, conn, path.Join(zkPath, child))
	}
	if err := conn.Delete(zkPath, -1); err != nil {
		t.Fatalf("error removing %q: %s", zkPath, err)
	}
}

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

	watcher, err := NewWatcher(conn)
	if err != nil {
		t.Fatal(err)
	}

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

	watcher, err := NewWatcher(conn)
	if err != nil {
		t.Fatal(err)
	}

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
