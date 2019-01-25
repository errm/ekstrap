export GO111MODULE=on

GOCMD=go
GOBUILD=$(GOCMD) build -ldflags="-s -w"
GOTEST=$(GOCMD) test
UPX=upx -9
BINARY_NAME=ekstrap
WORKDIR=/go/src/github.com/errm/ekstrap

GOMETALINTER = gometalinter ./...
GORELEASER = goreleaser release --rm-dist --debug

all: test lint $(BINARY_NAME)
$(BINARY_NAME): deps generate
	$(GOBUILD) -o $(BINARY_NAME) -v
compress: $(BINARY_NAME)
	strip -x $(BINARY_NAME)
	$(UPX) $(BINARY_NAME)
test: deps generate
	$(GOTEST) -coverprofile=coverage.txt -covermode=count ./...
install-linter:
	$(GOCMD) get -u github.com/alecthomas/gometalinter
	$(GOMETALINTER) --install
lint: deps
	$(GOMETALINTER)
release: generate .goreleaser.yml
	$(GORELEASER)
snapshot: .goreleaser.yml
	$(GORELEASER) --snapshot
install: $(BINARY_NAME)
	install -m755 $(BINARY_NAME) /usr/sbin
generate:
	$(GOCMD) generate
clean:
	$(GOCMD) clean
	rm -rf ./dist/
deps:
	$(GOCMD) build -v ./...
upgrade:
	$(GOCMD) get -u
