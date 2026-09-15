# syntax=docker/dockerfile:1

FROM golang:1.27.0-bookworm AS source
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY .github/workflows ./.github/workflows
COPY cmd ./cmd
COPY internal ./internal
COPY integration ./integration

FROM source AS test
RUN go test ./...
RUN go vet ./...
RUN go build ./...

FROM source AS build
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/dovik ./cmd/dovik
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/dovikd ./cmd/dovikd
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/gh ./cmd/gh

FROM debian:bookworm-slim AS runtime
ARG VERSION=1.0.0
ARG REVISION=unknown
LABEL org.opencontainers.image.title="Dovik" \
      org.opencontainers.image.description="Local development process supervisor" \
      org.opencontainers.image.source="https://github.com/MrMaxie/dovik" \
      org.opencontainers.image.licenses="Apache-2.0" \
      org.opencontainers.image.version="$VERSION" \
      org.opencontainers.image.revision="$REVISION"
RUN useradd --system --create-home --uid 10001 dovik
COPY --from=build /out/dovik /usr/local/bin/dovik
COPY --from=build /out/gh /usr/local/bin/gh
COPY --from=build /out/dovikd /usr/local/bin/dovikd
USER dovik
ENTRYPOINT ["/usr/local/bin/dovikd"]

FROM runtime AS integration
USER root
RUN apt-get update \
    && apt-get install --yes --no-install-recommends util-linux \
    && rm -rf /var/lib/apt/lists/*
USER dovik
