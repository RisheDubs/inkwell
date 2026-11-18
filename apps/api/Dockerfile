# syntax=docker/dockerfile:1

# ---- Stage 1: build the binary ----
# Runs on the build machine's own platform; Go cross-compiles for the target.
FROM --platform=$BUILDPLATFORM golang:1.27.3 AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src

# Download dependencies first, so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Copy the source and build a static, stripped binary for the target platform
# (linux/amd64 for local Compose, linux/arm64 for Fargate).
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- Stage 2: run it on a minimal, non-root base (pulled for the target platform) ----
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/api /api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]