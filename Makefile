GOCMD=go
GOBUILD=$(GOCMD) build -ldflags="-s -w"
GOTEST=$(GOCMD) test
UPX=upx --brute
BINARY_NAME=ekstrap
all: test build
build:
	$(GOBUILD) -o $(BINARY_NAME) -v
compress: build
	$(UPX) $(BINARY_NAME)
test:
	$(GOTEST) ./...
