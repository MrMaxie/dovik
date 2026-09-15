set dotenv-load := false

dev-client := if os_family() == "windows" { "build/dev/dovik.exe" } else { "build/dev/dovik" }
dev-daemon := if os_family() == "windows" { "build/dev/dovikd.exe" } else { "build/dev/dovikd" }

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
    mise exec -- go build -o {{dev-daemon}} ./cmd/dovikd
    mise exec -- go build -o {{dev-client}} ./cmd/dovik
    mise exec -- ./{{dev-client}} tui

tui-ttyglass:
    mise exec -- go build -o {{dev-client}} ./cmd/dovik
    mise exec -- node "{{justfile_directory()}}/devtools/ttyglass/node_modules/ttyglass/dist/cli.js" start --name "Dovik TUI" --cwd "{{justfile_directory()}}" -- "{{justfile_directory()}}/{{dev-client}}" tui

ttyglass-install:
    mise exec -- npm --prefix devtools/ttyglass ci --no-audit --no-fund

openspec-check:
    mise exec -- npx --yes @fission-ai/openspec@1.5.0 schema validate arcantry
    mise exec -- npx --yes @fission-ai/openspec@1.5.0 validate --all --strict --no-interactive

check: lint test build openspec-check

docs-install:
    mise exec -- npm --prefix docs/site ci --no-audit --no-fund

docs-check: docs-install
    mise exec -- npm --prefix docs/site audit --audit-level=high
    mise exec -- npm --prefix docs/site run check
    mise exec -- npm --prefix docs/site run build
    mise exec -- npm --prefix docs/site run check:links

docs-build: docs-install
    mise exec -- npm --prefix docs/site run build

license-check:
    mise exec -- go run ./cmd/releasepack check-licenses

package *args:
    mise exec -- go run ./cmd/releasepack package --output dist {{args}}

package-check: package

release-source-check: check docs-check license-check

test-linux:
    docker build --target test --tag dovik-test .

test-integration:
    mise exec -- go test -count=1 -tags=integration ./integration/...

test-identity-integration:
    mise exec -- go test -count=1 -tags=integration ./internal/identity/...

test-identity-interrupt:
    mise exec -- go build -o build/identity-interrupt/dovik{{ if os_family() == "windows" { ".exe" } else { "" } }} ./cmd/dovik
    mise exec -- go build -tags dovik_ttyglass -o build/identity-interrupt/dovikd{{ if os_family() == "windows" { ".exe" } else { "" } }} ./cmd/dovikd
    mise exec -- npm --prefix devtools/ttyglass run test:identity-interrupt

run-linux:
    docker build --target runtime --tag dovik:dev .
    docker run --rm dovik:dev
