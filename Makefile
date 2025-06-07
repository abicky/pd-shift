NAME := pd-shift
SRCS := $(shell find . -type f -name '*.go' -not -name '*_test.go')
BUILD_FLAGS := -trimpath -ldflags "-s -w -X github.com/abicky/pd-shift/cmd.revision=$(shell git rev-parse --short HEAD)"

all: bin/$(NAME)

bin/$(NAME): $(SRCS)
	go build -o $@ $(BUILD_FLAGS)

.PHONY: clean
clean:
	rm -rf bin/$(NAME)

.PHONY: install
install:
	go install $(BUILD_FLAGS)

.PHONY: test
test:
	go test -v ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: mock
mock:
	go generate ./...
