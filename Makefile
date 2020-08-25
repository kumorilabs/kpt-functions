.PHONY: all
all: build

.PHONY: build
build:
	@./build.sh

.PHONY: push
push:
	@./build.sh --push
