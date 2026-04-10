# Build stage - use build platform for faster compilation
FROM --platform=$BUILDPLATFORM docker.io/golang:1.25-alpine AS builder

# Build arguments automatically provided by Docker
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first for better caching
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the binary for target platform
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -ldflags='-w -s' \
    -o warehouse-service .

# Final stage - use target platform for runtime
FROM --platform=$TARGETPLATFORM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

# Copy CA certificates and timezone data from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy binary from builder
COPY --from=builder /app/warehouse-service /app/warehouse-service

# Distroless already runs as non-root user (uid 65532)
USER nonroot:nonroot

# Expose port
EXPOSE 11890

# Use absolute path in CMD
CMD ["/app/warehouse-service"]