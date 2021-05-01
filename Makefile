BUILD_GOOS ?= linux
BUILD_GOARCH ?= amd64
BUILD_CGO_ENABLED ?= 0

DOCKER_COMPOSE := docker-compose -f example/docker-compose.yml
DOCKER_CONTAINER_IMAGE="demdxx/asyncp-monitor:latest"
GO111MODULE := on

.PHONY: test
test: ## Run unit tests
	go test -race ./...

.PHONY: tidy
tidy: ## Apply tidy to the module
	go mod tidy

.PHONY: lint
lint:
	GL_DEBUG=true golangci-lint run -v ./...

.PHONY: mon-build
mon-build: ## Build monitor application
	@echo "Build monitor application"
	@rm -rf .build/apmonitor
	GOOS=${BUILD_GOOS} GOARCH=${BUILD_GOARCH} CGO_ENABLED=${BUILD_CGO_ENABLED} \
		go build -o .build/apmonitor cmd/apmonitor/main.go

.PHONY: mon-build-docker
mon-build-docker: mon-build ## Build monitor docker service
	echo "Build monitor docker image"
	docker build -t ${DOCKER_CONTAINER_IMAGE} -f docker/monitor.dockerfile .

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

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
