ALL_GO_FILES := $(shell find . -type f -name '*.go')

scream: $(ALL_GO_FILES)
	GOFLAGS=-mod=vendor go build .
