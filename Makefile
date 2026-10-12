.PHONY: help deps web build test lint notices check clean

RELEASE := go run ./cmd/release
NATIVE_ENV := eval "$$(scripts/native-deps env)"

help:
	@echo "make deps     fetch the pinned native layers for this machine"
	@echo "make web      install and build the web app"
	@echo "make build    build the release binary into dist/"
	@echo "make test     run go test -race -shuffle=on against the native layers"
	@echo "make lint     run the architecture lint, lock and notices checks"
	@echo "make notices  regenerate internal/notices/NOTICES.txt"
	@echo "make check    lint, test and build, as CI does"
	@echo "make clean    remove dist/ and web/dist/"

deps:
	scripts/native-deps fetch

web:
	cd web && bun install --frozen-lockfile && bun run build

build: deps
	$(RELEASE) build

test: deps
	$(NATIVE_ENV) && go test -race -shuffle=on ./...

lint: web
	$(RELEASE) lint

notices: web
	$(RELEASE) notices generate

check: lint test build

clean:
	rm -rf dist web/dist
