FROM golang:1.23-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download -x

COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION} -X main.GitCommit=${COMMIT}" \
    -o /bin/gateway ./cmd/gateway

FROM gcr.io/distroless/static-debian12:nonroot

LABEL maintainer="Staff Systems Architect <eng@wassup.chat>"
LABEL description="High-throughput distributed WebSocket chat gateway"

WORKDIR /app

COPY --from=builder --chown=nonroot:nonroot /bin/gateway /app/gateway

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/app/gateway"]
