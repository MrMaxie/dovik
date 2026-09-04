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

openspec-check:
    openspec schema validate arcantry
    openspec validate --all --strict --no-interactive

check: lint test build openspec-check

test-linux:
    docker build --target test --tag dovik-test .

test-integration:
    mise exec -- go test -count=1 -tags=integration ./integration/...

run-linux:
    docker build --target runtime --tag dovik:dev .
    docker run --rm dovik:dev
