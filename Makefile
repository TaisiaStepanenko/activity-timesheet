BINARY_NAME := atsheet.exe
BUILD_DIR := ./bin
MAIN_PKG := ./cmd/atsheet

.PHONY: all build test clean mod

all: mod build test

mod:
	go mod tidy

## Сборка бинарного файла
## -o задаёт имя выходного файла
build:
	go build -o ./$(BINARY_NAME) $(MAIN_PKG)

test:
	go test -race -cover ./...

clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	rm -f coverage.out