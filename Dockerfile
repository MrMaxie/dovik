# syntax=docker/dockerfile:1

FROM golang:1.27.0-bookworm AS source
WORKDIR /src
COPY . .

FROM source AS test
RUN go test ./...
RUN go vet ./...
RUN go build ./...

FROM source AS build
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/dovik ./cmd/dovik
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/dovikd ./cmd/dovikd

FROM debian:bookworm-slim AS runtime
RUN useradd --system --create-home --uid 10001 dovik
COPY --from=build /out/dovik /usr/local/bin/dovik
COPY --from=build /out/dovikd /usr/local/bin/dovikd
USER dovik
ENTRYPOINT ["/usr/local/bin/dovikd"]

FROM runtime AS integration
USER root
RUN apt-get update \
    && apt-get install --yes --no-install-recommends util-linux \
    && rm -rf /var/lib/apt/lists/*
USER dovik
