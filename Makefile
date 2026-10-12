.PHONY: help deps web build test lint fmt notices check clean

RELEASE := go run ./cmd/release

help:
	@echo "make deps     fetch the pinned native layers for this machine"
	@echo "make web      install and build the web app"
	@echo "make build    build the release binary into dist/"
	@echo "make test     run go test -race -shuffle=on against the native layers"
	@echo "make lint     run the Go and web linters, format, architecture and notices checks"
	@echo "make fmt      format the Go code and the web app"
	@echo "make notices  regenerate internal/notices/NOTICES.txt"
	@echo "make check    lint, test and build, as CI does"
	@echo "make clean    remove dist/ and web/dist/"

deps:
	$(RELEASE) deps fetch

web:
	$(RELEASE) web

build:
	$(RELEASE) build

test:
	$(RELEASE) test

lint:
	$(RELEASE) lint

fmt:
	$(RELEASE) fmt

notices:
	$(RELEASE) notices generate

check:
	$(RELEASE) check

clean:
	rm -rf dist web/dist
