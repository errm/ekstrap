GOCMD=go
GOBUILD=$(GOCMD) build -ldflags="-s -w"
GOTEST=$(GOCMD) test
UPX=upx --brute
BINARY_NAME=ekstrap
WORKDIR=/go/src/github.com/errm/ekstrap

GOMETALINTER = gometalinter ./...
GORELEASER = goreleaser release --rm-dist --debug

all: test lint $(BINARY_NAME)
$(BINARY_NAME):
	$(GOBUILD) -o $(BINARY_NAME) -v
compress: $(BINARY_NAME)
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
install: $(BINARY_NAME)
	install -m755 $(BINARY_NAME) /usr/sbin

clean:
	rm -rf \
		./$(BINARY_NAME) \
		./dist/
