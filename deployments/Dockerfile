# Stage 1: Build binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compile statically linked binary with stripped debug symbols
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w -extldflags '-static'" \
    -o /app/bin/gateway \
    ./cmd/gateway/main.go

# Stage 2: Minimal runtime using unprivileged Alpine user
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata

# Use standard non-root user (nobody:nobody: UID 65534)
USER 65534:65534

WORKDIR /home/nobody

COPY --from=builder --chown=65534:65534 /app/bin/gateway /usr/local/bin/gateway

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/gateway"]
