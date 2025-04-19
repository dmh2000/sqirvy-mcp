# pkg directory Makefile

.PHONY: all build test clean

SUBDIRS := transport types utils

all: build test

build:
	@echo "Building pkg..."
	@for dir in $(SUBDIRS); do \
		$(MAKE) -C $$dir build; \
	done

test:
	@echo "Testing pkg..."
	@for dir in $(SUBDIRS); do \
		$(MAKE) -C $$dir test; \
	done

clean:
	@echo "Cleaning pkg..."
	@for dir in $(SUBDIRS); do \
		$(MAKE) -C $$dir clean; \
	done
