BINARY_NAME=env-run
BUILD_DIR=build

.PHONY: all build test clean

all: test build

build:
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) .

test:
	go test -v ./...

clean:
	rm -rf $(BUILD_DIR)
