package zkwatcher_test

import (
	"fmt"
	"log"
	"time"

	"github.com/agrarianlabs/zkwatcher"
	"github.com/samuel/go-zookeeper/zk"
)

func Example() {
	conn, _, err := zk.Connect([]string{"zookeeper host addr"}, 10*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	watcher, err := zkwatcher.NewWatcher(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = watcher.Close() }()

	// Make sure the path exists before starting the watcher.
	if err := watcher.Watch("/path/to/watch"); err != nil {
		log.Fatal(err)
	}

	for event := range watcher.C {
		fmt.Printf("%s on %s\n", event.Type, event.Path)
	}
}
