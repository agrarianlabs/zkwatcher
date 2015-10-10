# ZKWatcher

ZKWatcher provides an easy way to recursively watch a zookeeper path and get meaningful events out of it.

## Dependencies

The only dependency is the samuel's zookeeper driver: `github.com/samuel/go-zookeeper/zk`.
If you have `godep`, you can use it directly, otherwise, you might need to download the driver: `go get -d`.

## Usage

### Example

```go
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/agrarianlabs/zkwatcher"
	"github.com/samuel/go-zookeeper/zk"
)

func main() {
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
```

## Tests

The tests are only integration tests and relies on a running zookeeper.
The tests will create random roots prefixed with `zkwatchertest_`.
Even in case of failure, those should be removed.

### With Docker

If you have Docker, make sure you have the `DOCKER_IP` variable defined and run:

```bash
make test
```

### Without Docker

If you don't have Docker, you can spin up a Zookeeper on your own and then run:

```bash
go test -v -tags integration -zk-host $YOUR_ZK_TARGET .
```

or if you have godep:

```bash
godep go test -v -tags integration -zk-host $YOUR_ZK_TARGET .
```
