# transport directory Makefile

.PHONY: all build test clean

GO_FILES := $(wildcard *.go)
TEST_FILES := $(wildcard *_test.go)

all: build test

build:
	@echo "Building transport..."
	@staticcheck .

test:
	@echo "Testing transport..."
	@if [ "$(TEST_FILES)" != "" ]; then \
		go test -v .; \
	fi

clean:
	@echo "Cleaning transport..."
	@rm -f *.test
