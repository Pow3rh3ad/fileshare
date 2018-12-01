
BINARY := fileshare
PLATFORMS := windows linux darwin
VERSION ?= vlatest
os = $(word 1, $@)
PKGS := $(shell go list ./... | grep -v /vendor)
BIN_DIR := $(GOPATH)/bin

.PHONY: test
test: 
	go test $(PKGS) -v

.PHONY: $(PLATFORMS)
$(PLATFORMS):
	mkdir -p release
	GOOS=$(os) GOARCH=amd64 go build -o release/$(BINARY)-v1.0.0-$(os)-amd64

.PHONY: release
release: windows linux darwin
