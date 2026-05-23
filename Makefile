.PHONY: app build add list remove edit help

BIN := bin/kura

build:
	go build -o $(BIN) ./cmd/app

app: build
	./$(BIN)

# Subcommands.
# Usage:
#   make add name=foo
#   make list
#   make remove name=foo            # stdin 'yes' to confirm
#   make edit old=foo new=bar
#   make help
add:
	go run ./cmd/app add $(name)

list:
	go run ./cmd/app list

remove:
	go run ./cmd/app remove $(name)

edit:
	go run ./cmd/app edit $(old) $(new)

help:
	go run ./cmd/app help
