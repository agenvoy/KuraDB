.PHONY: app build stop test add list remove edit help

BIN := bin/kura

ifneq (,$(wildcard .env))
include .env
export
endif

build:
	go build -o $(BIN) ./cmd/app

app:
	@$(MAKE) stop
	@$(MAKE) build
	@sudo mkdir -p /usr/local/bin && sudo mv $(BIN) /usr/local/bin/kura
	@kura

stop:
	@go run ./cmd/app stop

test:
	@go test -v -count=1 ./...

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
