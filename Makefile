BUILD_GOOS ?= $(or ${DOCKER_DEFAULT_GOOS},linux)
BUILD_GOARCH ?= $(or ${DOCKER_DEFAULT_GOARCH},amd64)
BUILD_GOARM ?= 7
BUILD_CGO_ENABLED ?= 0

DOCKER_COMPOSE := docker-compose -f example/docker-compose.yml
DOCKER_CONTAINER_IMAGE="demdxx/asyncp-monitor:latest"
GO111MODULE := on
DOCKER_BUILDKIT := 1

.PHONY: test
test: ## Run unit tests
	go test -race ./...

.PHONY: tidy
tidy: ## Apply tidy to the module
	go mod tidy

.PHONY: lint
lint:
	GL_DEBUG=true golangci-lint run -v ./...

OS_LIST = linux darwin
ARCH_LIST = amd64 arm64 arm

.PHONY: mon-build
mon-build: ## Build monitor application
	@echo "Build monitor application"
	@rm -rf .build/apmonitor
	for os in $(OS_LIST); do \
		for arch in $(ARCH_LIST); do \
			echo "Build $$os/$$arch"; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=${BUILD_CGO_ENABLED} \
				go build -o .build/$$os/$$arch/apmonitor cmd/apmonitor/main.go; \
		done \
	done

.PHONY: mon-build-docker
mon-build-docker: mon-build ## Build monitor docker service
	echo "Build monitor docker image"
	# --cache-from type=local,dest=/tmp/docker-cache
	# --cache-to   type=local,dest=/tmp/docker-cache,mode=max
	DOCKER_BUILDKIT=${DOCKER_BUILDKIT} docker buildx build \
		--push --platform linux/amd64,linux/arm64,darwin/amd64,darwin/arm64 \
		-t ${DOCKER_CONTAINER_IMAGE} -f docker/monitor.dockerfile .

mon-build-docker-dev:
	echo "Build monitor docker image for dev"
	GOOS=${BUILD_GOOS} GOARCH=${BUILD_GOARCH} CGO_ENABLED=${BUILD_CGO_ENABLED} GOARM=${BUILD_GOARM} \
		go build -o .build/apmonitor cmd/apmonitor/main.go
	DOCKER_BUILDKIT=${DOCKER_BUILDKIT} docker build \
		-t ${DOCKER_CONTAINER_IMAGE} -f docker/monitor-dev.dockerfile .

.PHONY: run-monitor-test
run-monitor-test:
	rm -fR .build/
	$(DOCKER_COMPOSE) stop
	$(DOCKER_COMPOSE) rm -f counter
	GOOS=${BUILD_GOOS} GOARCH=${BUILD_GOARCH} CGO_ENABLED=${BUILD_CGO_ENABLED} GOARM=${BUILD_GOARM} \
		go build -o .build/counter example/counter/main.go
	GOOS=${BUILD_GOOS} GOARCH=${BUILD_GOARCH} CGO_ENABLED=${BUILD_CGO_ENABLED} GOARM=${BUILD_GOARM} \
		go build -o .build/apmonitor cmd/apmonitor/main.go
	$(DOCKER_COMPOSE) run --rm --service-ports apmonitor

.PHONY: fmt
fmt: ## Run formatting code
	@echo "Fix formatting"
	@gofmt -w ${GO_FMT_FLAGS} $$(go list -f "{{ .Dir }}" ./...); if [ "$${errors}" != "" ]; then echo "$${errors}"; fi

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
