# Makefile for Go project

# Variables
GOCMD = go
GOBUILD = $(GOCMD) build
GOTEST = $(GOCMD) test
BINARY_NAME = m3uproxy
BUILD_DIR = build

# Targets and Commands
all: clean build test

build: build-server

build-server:
	mkdir -p ${BUILD_DIR}
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) -v ./server

test:
	$(GOTEST) -v ./pkg/...

clean:
	rm -rf $(BUILD_DIR)
