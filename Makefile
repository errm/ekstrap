export GO111MODULE=on

GOCMD=go
GOBUILD=$(GOCMD) build -ldflags="-s -w"
GOTEST=$(GOCMD) test
UPX=upx -9
BINARY_NAME=ekstrap
PWD=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

GORELEASER = goreleaser release --rm-dist --debug

all: test lint $(BINARY_NAME)
$(BINARY_NAME): deps generate
	$(GOBUILD) -o $(BINARY_NAME) -v
compress: $(BINARY_NAME)
	strip -x $(BINARY_NAME)
	$(UPX) $(BINARY_NAME)
test: deps generate
	$(GOTEST) -coverprofile=coverage.txt -covermode=count ./...
lint:
	command -v golangci-lint || GO111MODULE=off $(GOCMD) get -u github.com/golangci/golangci-lint/cmd/golangci-lint
	golangci-lint run
release: generate .goreleaser.yml
	$(GORELEASER)
snapshot: .goreleaser.yml
	$(GORELEASER) --snapshot
install: $(BINARY_NAME)
	install -m755 $(BINARY_NAME) /usr/sbin
generate:
	command -v packr2 || $(GOCMD) get github.com/gobuffalo/packr/v2/packr2@v2.0.2
	$(GOCMD) generate
clean:
	$(GOCMD) clean
	packr2 clean
	rm -rf ./dist/
deps:
	$(GOCMD) build -v ./...
upgrade:
	$(GOCMD) get -u
update-instance-types:
	ruby pkg/node/resources.rb
