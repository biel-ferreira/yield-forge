# syntax=docker/dockerfile:1

# ---- Build stage ----
# Go 1.24 is the module's minimum (SPEC-002 pulled in pgx and its deps).
FROM golang:1.24-alpine AS build

WORKDIR /src

# Download dependencies first for better layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the source and build a static, stripped binary. Build metadata is injected
# via linker flags (overridable with --build-arg).
COPY . .

ARG VERSION=dev
ARG COMMIT=none
ARG BUILD_TIME=unknown
ENV CGO_ENABLED=0 GOOS=linux

RUN go build \
    -trimpath \
    -ldflags="-s -w \
      -X github.com/biel-ferreira/yield-forge/internal/platform/buildinfo.Version=${VERSION} \
      -X github.com/biel-ferreira/yield-forge/internal/platform/buildinfo.Commit=${COMMIT} \
      -X github.com/biel-ferreira/yield-forge/internal/platform/buildinfo.BuildTime=${BUILD_TIME}" \
    -o /out/yield-forge ./cmd/api

# ---- Runtime stage ----
# distroless/static:nonroot — no shell, no package manager, runs as a non-root
# user (uid 65532). Minimal attack surface for a static Go binary.
FROM gcr.io/distroless/static:nonroot

WORKDIR /app
COPY --from=build /out/yield-forge /app/yield-forge

EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/yield-forge"]
