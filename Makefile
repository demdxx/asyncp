BUILD_GOOS ?= linux
BUILD_GOARCH ?= amd64
BUILD_CGO_ENABLED ?= 0

DOCKER_COMPOSE := docker-compose -f example/docker-compose.yml
GO111MODULE := on

.PHONY: test
test:
	go test -race ./...

.PHONY: lint
lint:
	GL_DEBUG=true golangci-lint run -v ./...

.PHONY: run-monitor-test
run-monitor-test:
	rm -fR .build/
	$(DOCKER_COMPOSE) stop
	$(DOCKER_COMPOSE) rm -f counter
	GOOS=${BUILD_GOOS} GOARCH=${BUILD_GOARCH} CGO_ENABLED=${BUILD_CGO_ENABLED} \
		go build -o .build/counter example/counter/main.go
	GOOS=${BUILD_GOOS} GOARCH=${BUILD_GOARCH} CGO_ENABLED=${BUILD_CGO_ENABLED} \
		go build -o .build/apmonitor cmd/apmonitor/main.go
	$(DOCKER_COMPOSE) run --rm --service-ports apmonitor

.PHONY: docker-stop
docker-stop:
	$(DOCKER_COMPOSE) stop

.PHONY: redis-cli
redis-cli:
	$(DOCKER_COMPOSE) exec redis redis-cli
