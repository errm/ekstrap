GOCMD=go
GOBUILD=$(GOCMD) build -ldflags="-s -w"
GOTEST=$(GOCMD) test
UPX=upx --brute
BINARY_NAME=ekstrap
WORKDIR=/go/src/github.com/errm/ekstrap

GOMETALINTER = gometalinter ./...
GORELEASER = goreleaser release --rm-dist --debug

all: test lint build
build:
	$(GOBUILD) -o $(BINARY_NAME) -v
compress: build
	$(UPX) $(BINARY_NAME)
test:
	$(GOTEST) -coverprofile=coverage.txt -covermode=count ./...
install-linter:
	$(GOCMD) get -u github.com/alecthomas/gometalinter
	$(GOMETALINTER) --install
lint:
	$(GOMETALINTER)
release: .goreleaser.yml
	$(GORELEASER)
snapshot: .goreleaser.yml
	$(GORELEASER) --snapshot

clean:
	rm -rf \
		./ekstrap \
		./dist/
