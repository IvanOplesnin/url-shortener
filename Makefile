SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c

.PHONY: run test run_memory

run:
	ENV_FILE=./.env ./run.sh

test:
	go test ./... -v


run_memory:
	ENV_FILE=./.env.inmemory ./run.sh
