# types directory Makefile

.PHONY: all build test clean

GO_FILES := $(wildcard *.go)
TEST_FILES := $(wildcard *_test.go)

all: build test

build:
	@echo "Building types..."
	@staticcheck .

test:
	@echo "Testing types..."
	@if [ "$(TEST_FILES)" != "" ]; then \
		go test -v .; \
	fi

clean:
	@echo "Cleaning types..."
	@rm -f *.test
