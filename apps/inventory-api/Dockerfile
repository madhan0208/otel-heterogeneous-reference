# syntax=docker/dockerfile:1.6

# ----- Build stage --------------------------------------------------
FROM golang:1.25-alpine AS build
WORKDIR /src

# Copy module files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build a static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/inventory-api -ldflags="-s -w" .

# ----- Runtime stage ------------------------------------------------
# distroless/static: no shell, no libc, non-root user by default
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/inventory-api /usr/local/bin/inventory-api

EXPOSE 8081
USER nonroot:nonroot
ENTRYPOINT ["/usr/local/bin/inventory-api"]