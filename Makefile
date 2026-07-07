.PHONY: app build stop test add list remove edit port help

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
# Usage
#   make add foo
#   make list
#   make remove foo                 # stdin 'yes' to confirm
#   make edit old new
#   make port set 8080 | make port clear
#   make help
SUBCOMMANDS := add list remove edit port
ifneq (,$(filter $(firstword $(MAKECMDGOALS)),$(SUBCOMMANDS)))
ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
$(eval $(ARGS):;@:)
endif

add:
	go run ./cmd/app add $(ARGS)

list:
	go run ./cmd/app list

remove:
	go run ./cmd/app remove $(ARGS)

edit:
	go run ./cmd/app edit $(ARGS)

port:
	go run ./cmd/app port $(ARGS)

help:
	go run ./cmd/app help
