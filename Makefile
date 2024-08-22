INSTALL_DIR = /usr/bin
BINARY_NAME = gart
VERSION := $(shell git describe --tags --always --dirty || echo 'unknown')
COMMIT := $(shell git rev-parse --short HEAD)
DATE := $(shell date -u +"%Y-%m-%d %H:%M:%S")
PACKAGE := github.com/bnema/gart/internal/version

# Default target
all: build

# Build the Go binary
build:
	go build -ldflags " \
		-X '$(PACKAGE).Version=$(VERSION)' \
		-X '$(PACKAGE).Commit=$(COMMIT)' \
		-X '$(PACKAGE).Date=$(DATE)'" \
		-o $(BINARY_NAME) .

# Install the binary to /usr/local/bin
install:
	mv $(BINARY_NAME) $(INSTALL_DIR)

# Clean up the built binary
clean:
	rm -f $(BINARY_NAME)

.PHONY: all build install clean
