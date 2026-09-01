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

openspec-check:
    openspec schema validate arcantry
    openspec validate --all --strict --no-interactive

check: lint test build openspec-check

test-linux:
    docker build --target test --tag dovik-test .

run-linux:
    docker build --target runtime --tag dovik:dev .
    docker run --rm dovik:dev
