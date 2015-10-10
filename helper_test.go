package zkwatcher

import (
	"flag"
	"math/rand"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/samuel/go-zookeeper/zk"
)

var testZKHost string

// init initializes the random seed and pull the zk-host from command line.
func init() {
	rand.Seed(time.Now().UTC().UnixNano())

	// Get the testing zookeeper host from command line.
	flag.StringVar(&testZKHost, "zk-host", "comma separated testing zookeeper hosts", "<none>")

}

// randomName generates a random string prefixed with "zkwatchtest".
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
