MAIN_NAME=container
export GOPROXY=https://goproxy.io,direct
export CGO_ENABLED := 0
export GO=go
PACKAGES = $(shell go list ./... | grep -v /vendor/)
GOFILES=`find . -name "*.go" -type f -not -path "./vendor/*"`

all: help

build: ## build container
	${GO} build -o ${MAIN_NAME} cmd/containerd/*.go

dev: build
	sudo ./${MAIN_NAME} run alpine sh

.PHONY: vet
vet: ## vet
	@ go vet ${PACKAGES}

.PHONY: help
help: ## help
	@grep -E '^[ a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'
