# syntax=docker/dockerfile:1

# ---- Stage 1: build the binary ----
FROM golang:1.27.3 AS build
WORKDIR /src

# Download dependencies first, so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN go mod download

# Copy the source and build a static, stripped binary.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

# ---- Stage 2: run it on a minimal, non-root base ----
FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/api /api
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]