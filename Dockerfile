# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE} -s -w" \
    -o /schema-registry ./cmd/schema-registry

# Runtime stage
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 schemaregistry && \
    adduser -u 1000 -G schemaregistry -s /bin/sh -D schemaregistry

WORKDIR /app

# Copy binary from builder
COPY --from=builder /schema-registry /app/schema-registry

# Copy default config (optional)
# COPY config.yaml /app/config.yaml

# Set ownership
RUN chown -R schemaregistry:schemaregistry /app

USER schemaregistry

EXPOSE 8081

ENTRYPOINT ["/app/schema-registry"]
