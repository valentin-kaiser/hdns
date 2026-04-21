# Retrieve git information and set the version variables via ldflags for the build
GIT_TAG := $(shell git describe --tags)
GIT_COMMIT := $(shell git rev-parse HEAD)
GIT_SHORT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date +%FT%T%z)
PACKAGE := github.com/Valentin-Kaiser/go-core/version
LDFLAGS := -ldflags "-X $(PACKAGE).GitTag=$(GIT_TAG) -X $(PACKAGE).GitCommit=$(GIT_COMMIT) -X $(PACKAGE).GitShort=$(GIT_SHORT) -X $(PACKAGE).BuildDate=$(BUILD_TIME)"

RELEASE_FILE=../../../hdns
ifeq ($(OS),Windows_NT)
	RELEASE_FILE:=$(RELEASE_FILE).exe
endif

prod:
	go build $(LDFLAGS) -o $(RELEASE_FILE)
lint:
	golangci-lint run
run:
	go run $(LDFLAGS) main.go --debug
