# utils directory Makefile

.PHONY: all build test clean

GO_FILES := $(wildcard *.go)
TEST_FILES := $(wildcard *_test.go)

all: build test

build:
	@echo "Building utils..."
	@staticcheck .

test:
	@echo "Testing utils..."
	@if [ "$(TEST_FILES)" != "" ]; then \
		go test .; \
	fi

clean:
	@echo "Cleaning utils..."
	@rm -f *.test
