GOCMD=go
GOBUILD=$(GOCMD) build -ldflags="-s -w"
GOTEST=$(GOCMD) test
UPX=upx --brute
BINARY_NAME=ekstrap
WORKDIR=/go/src/github.com/errm/ekstrap

DOCKER = /usr/bin/env docker
GORELEASER_VERSION = 0.77.2
GORELEASER_BUILD = $(DOCKER) build --rm -f Dockerfile.release --build-arg GORELEASER_VERSION=$(GORELEASER_VERSION) -t ekstrap-release:$(GORELEASER_VERSION) .
GORELEASER = $(DOCKER) run --rm --workdir $(WORKDIR) --volume $$(pwd):$(WORKDIR) ekstrap-release:$(GORELEASER_VERSION)

all: test build
build:
	$(GOBUILD) -o $(BINARY_NAME) -v
compress: build
	$(UPX) $(BINARY_NAME)
test:
	$(GOTEST) ./...

build-releaser: Dockerfile.release
	$(GORELEASER_BUILD)
release: build-releaser .goreleaser.yml
	$(GORELEASER) release --rm-dist
snapshot: build-releaser .goreleaser.yml
	$(GORELEASER) release --rm-dist --snapshot --debug
