ZK_IMAGE        = jplock/zookeeper:latest
ZK_CONTAINER    = zkwatchertest_c
ZK_HOST         = $(DOCKER_IP)
ZK_PORT         = 2181
GO              = $(shell hash godep 2> /dev/null && echo "godep go" || echo "go")
MAKE           ?= make
OPTS           ?= -v -cover -covermode=count -coverprofile=coverprofile

docker_ip       :
		@[ -n "$(DOCKER_IP)" ] || (echo "Please run `export DOCKER_IP=<target docker ip>`."; exit 1)

# TODO: implement better way to detect that zookeeper is up and running.
start_zookeeper : docker_ip
		@docker run -d -p $(ZK_PORT) --name $(ZK_CONTAINER) $(ZK_IMAGE) > /dev/null
		@sleep 1
		@docker port $(ZK_CONTAINER) $(ZK_PORT) | sed 's/.*://' > .zk_port

stop_zookeeper  : docker_ip
		@docker rm -f $(ZK_CONTAINER) > /dev/null 2> /dev/null || true
		@rm -f .zk_port

_test           : start_zookeeper
		@$(GO) test $(OPTS) -tags integration -zk-host=$(DOCKER_IP):$(shell cat .zk_port) || ($(MAKE) stop_zookeeper; exit 1)

test            : _test stop_zookeeper

clean           : stop_zookeeper
		@rm -f coverprofile zkwatcher.test

.PHONY          : start_zookeeper stop_zookeeper test _test clean
