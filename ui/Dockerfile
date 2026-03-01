# ── Stage 1: Build React SPA ──
FROM node:22-alpine AS frontend

WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

COPY . .
RUN npm run build

# ── Stage 2: Build Go binary ──
FROM golang:1.26-alpine AS backend

WORKDIR /app

RUN apk add --no-cache git ca-certificates

# Copy Go module files and download deps
COPY go.mod go.sum ./
RUN go mod download

# Copy Go source
COPY cmd/ cmd/
COPY internal/ internal/
COPY web/ web/

# Copy built frontend into the embed directory
COPY --from=frontend /app/dist/ web/dist/

# Build binary
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.buildDate=${BUILD_DATE} -s -w" \
    -o /schema-registry-ui ./cmd/schema-registry-ui

# ── Stage 3: Runtime ──
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1000 srui && \
    adduser -u 1000 -G srui -s /bin/sh -D srui

WORKDIR /app

COPY --from=backend /schema-registry-ui /app/schema-registry-ui

# Default htpasswd directory
RUN mkdir -p /etc/schema-registry-ui && \
    chown -R srui:srui /app /etc/schema-registry-ui

USER srui

EXPOSE 8080

ENTRYPOINT ["/app/schema-registry-ui"]
