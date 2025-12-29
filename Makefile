SHELL := /usr/bin/env bash
.SHELLFLAGS := -eu -o pipefail -c

.PHONY: run test

run:
	./run.sh

test:
	go test ./... -v
