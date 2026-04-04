.PHONY: all build build-mac build-linux build-all clean install

VERSION ?= dev
BINARY_NAME := poyo
BUILD_DIR := build

# Go parameters
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# Go proxy for faster downloads
GOPROXY ?= https://goproxy.cn,direct

# Build flags
LDFLAGS := -s -w -X main.Version=$(VERSION)

all: clean build

build:
	GOPROXY=$(GOPROXY) $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/poyo

build-mac:
	GOPROXY=$(GOPROXY) GOOS=darwin GOARCH=arm64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/poyo

build-mac-intel:
	GOPROXY=$(GOPROXY) GOOS=darwin GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/poyo

build-linux:
	GOPROXY=$(GOPROXY) GOOS=linux GOARCH=amd64 $(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/poyo

build-all: build-mac build-mac-intel build-linux
	echo "All builds complete!"

clean:
	rm -rf $(BUILD_DIR)
	$(GOCLEAN)

install: build
	cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

test:
	$(GOTEST) -v ./...

deps:
	GOPROXY=$(GOPROXY) $(GOMOD) download
	$(GOMOD) tidy

# Development targets
dev:
	$(GOBUILD) -race -o $(BUILD_DIR)/$(BINARY_NAME)-debug ./cmd/poyo

fmt:
	$(GOCMD) fmt ./...

lint:
	golangci-lint run ./...
