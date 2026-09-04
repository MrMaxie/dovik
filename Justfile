set dotenv-load := false

default:
    @just --list

build:
    mise exec -- go build ./...

test:
    mise exec -- go test ./...

format:
    mise exec -- go fmt ./...
    mise exec -- goimports -w .

lint:
    mise exec -- go vet ./...

run *args:
    mise exec -- go run ./cmd/dovikd {{args}}

run-cli *args:
    mise exec -- go run ./cmd/dovik {{args}}

tui:
    mise exec -- go run ./cmd/dovik tui

tui-web:
    mise exec -- npm --prefix devtools/tui-harness install --no-audit --no-fund
    mise exec -- node devtools/tui-harness/server.mjs

openspec-check:
    openspec schema validate arcantry
    openspec validate --all --strict --no-interactive

test-tui-web:
    mise exec -- npm --prefix devtools/tui-harness install --no-audit --no-fund
    mise exec -- npm --prefix devtools/tui-harness test

check: lint test build test-tui-web openspec-check

test-linux:
    docker build --target test --tag dovik-test .

test-integration:
    mise exec -- go test -count=1 -tags=integration ./integration/...

run-linux:
    docker build --target runtime --tag dovik:dev .
    docker run --rm dovik:dev
