# Wassup

Wassup is a Go-based WebSocket messaging gateway with a small browser demo. The gateway authenticates clients with PASETO v4.public tokens, accepts messaging envelopes, and routes messages to connected users. Redis and MongoDB adapters provide the building blocks for session lookup, rate limiting, pub/sub, and offline message persistence.

The included web page is a protocol demo. It Base64-encodes typed message text to fit the gateway's ciphertext payload format; it does **not** perform end-to-end encryption.

## Features

- Authenticated WebSocket connections at `/ws`.
- Redis adapters for session lookup, distributed rate limiting, and pub/sub.
- Offline message persistence in MongoDB.
- Per-IP and per-user connection rate limiting.
- Health, readiness, and Prometheus metrics endpoints.
- Nginx-served browser demo and local development launcher.

## Architecture

```text
Browser clients
      |
      | HTTP / WebSocket :8080
      v
Nginx (static UI and reverse proxy)
      |                         \
      | host.docker.internal    \ static files
      v                           v
Gateway :8081 / :8082           web/
      |             |
      +------ Redis +------ MongoDB
```

For local development, Docker Compose starts Nginx, Redis, and MongoDB. The Go gateway processes run on the host and listen on ports `8081` and `8082`; Nginx forwards WebSocket and API traffic to them.

## Requirements

- Go **1.27.1 or newer** (matching the `go` version in `go.mod`).
- Docker Desktop with Docker Compose, or Docker Engine and the Compose plugin.
- Windows for the supplied `run.bat` launcher. On other operating systems, start the dependencies and gateway processes manually.

## Run locally on Windows

1. Clone the repository and open a terminal in the project directory.
2. Generate a demo signing key and tokens:

   ```powershell
   go run ./cmd/keygen
   ```

   The command prints `PASETO_PUBLIC_KEY_HEX` and one token each for `user_alice` and `user_bob`. The signing key exists only for that keygen run; save the printed tokens if you need them. Tokens expire after 24 hours.

3. Put the generated public key value into the `PASETO_PUBLIC_KEY_HEX` setting in **both gateway commands** in `run.bat`. Both gateway nodes must use the same key that signed the tokens.
4. Start the local stack:

   ```powershell
   .\run.bat
   ```

   This starts Redis, MongoDB, and Nginx with Docker Compose, then opens two gateway processes. Nginx and the browser demo are available at [http://localhost:8080](http://localhost:8080).

5. Open the page in two browser tabs:
   - Tab 1: sender `user_alice`, Alice's token, recipient `user_bob`.
   - Tab 2: sender `user_bob`, Bob's token, recipient `user_alice`.
   - Use the same conversation ID in both tabs, such as `conv_alice_bob`.
6. Connect both tabs, then send a message from either one.

If you generate a new key, update `run.bat` and restart both gateways before using the matching tokens. The public key and tokens must always come from the same keypair.

To stop the gateways, close their command windows. Stop the Docker services with:

```powershell
docker compose down
```

## Run components manually

Start infrastructure and the browser proxy:

```sh
docker compose up -d redis mongo loadbalancer
```

Set `PASETO_PUBLIC_KEY_HEX` to the public key from `go run ./cmd/keygen`, then start each gateway in its own terminal with the same key and local service addresses:

```sh
SERVER_PORT=8081 NODE_ID=node-gw-01 \
REDIS_ADDR=localhost:6379 \
MONGO_URI=mongodb://localhost:27017/wassup_chat \
MONGO_DATABASE=wassup_chat \
PASETO_PUBLIC_KEY_HEX=<public-key-hex> \
go run ./cmd/gateway/main.go
```

Run the second node on port `8082` with a different `NODE_ID`. On Windows PowerShell, set environment variables with `$env:NAME = "value"` before running the command. `run.bat` is the ready-to-use Windows path.

## WebSocket protocol

Connect to `ws://localhost:8080/ws?token=<PASETO>` through the local Nginx proxy. The browser demo uses the query parameter because the browser WebSocket API does not support setting an Authorization header. In other clients, the gateway also accepts `Authorization: Bearer <PASETO>`.

A message envelope has this shape:

```json
{
  "idempotency_id": "unique-message-id",
  "conversation_id": "conv_alice_bob",
  "type": "MESSAGE",
  "sender_id": "user_alice",
  "recipient_id": "user_bob",
  "seq_id": 1,
  "timestamp": 1793120000000,
  "payload": {
    "ciphertext": "aGVsbG8=",
    "cipher_suite": "",
    "key_id": "",
    "nonce": ""
  },
  "trace_id": ""
}
```

`payload.ciphertext` and `payload.nonce` are JSON Base64 strings because the corresponding Go fields are byte slices. A client acknowledgment uses type `CLIENT_ACK` and places the acknowledged message ID in `payload.ciphertext` (Base64-encoded). The gateway sets the authenticated user as the sender for inbound messages.

## Configuration

The gateway reads configuration from environment variables:

| Variable | Purpose | Default |
| --- | --- | --- |
| `SERVER_PORT` | Gateway HTTP/WebSocket listen port | `8080` |
| `REDIS_ADDR` | Redis address | `127.0.0.1:6379` |
| `REDIS_USERNAME` | Redis ACL username | Empty |
| `REDIS_PASSWORD` | Redis password | Empty |
| `REDIS_DB` | Redis database number | `0` |
| `MONGO_URI` | MongoDB connection URI | `mongodb://localhost:27017/wassup_chat` |
| `MONGO_DATABASE` | MongoDB database name | `wassup_chat` |
| `PASETO_PUBLIC_KEY_HEX` | 32-byte Ed25519 public key as 64 hex characters | Required |

Additional Redis pool and timeout settings are available in `internal/adapter/secondary/redis/client.go`.

## HTTP endpoints

| Path | Purpose |
| --- | --- |
| `/ws` | Authenticated WebSocket upgrade endpoint |
| `/healthz` | Liveness check |
| `/readyz` | Readiness check (gateway directly) |
| `/metrics` | Prometheus metrics |

The local Nginx configuration proxies `/ws`, `/healthz`, and `/metrics`; query `/readyz` directly on a gateway, for example `http://localhost:8081/readyz`.

## Build

```sh
go build ./cmd/gateway
go build ./cmd/keygen
```

The repository also includes Kubernetes and container deployment manifests under `deployments/`. Review their image versions, secrets, networking, and environment configuration for your target environment before deploying.

### Implementation notes

- The gateway currently routes directly to sessions connected to its local hub. Redis session-registry and pub/sub components exist, but active WebSocket sessions are not yet wired into a complete cross-node delivery path. Treat multi-node delivery as in progress.
- The container build file currently uses a Go 1.23 builder image, while `go.mod` requires Go 1.27.1. Update the builder image before building the gateway container.

## Project layout

```text
cmd/gateway/                       Gateway entry point
cmd/keygen/                        Demo PASETO key and token generator
internal/adapter/handler/websocket WebSocket handler, sessions, and hub
internal/adapter/secondary/        Redis and MongoDB adapters
internal/core/                     Domain types, ports, and message routing
internal/security/                 PASETO verification and rate limiting
deployments/                       Nginx, container, and Kubernetes configs
test/                               Integration and load-test assets
web/                                Browser demo
```

## Security notes

- Keep signing private keys and real user tokens out of source control. The key generator included here is for local demonstration; it does not persist the private key or provide a production token-issuance service.
- Use TLS (`wss://`) and protect tokens in production. Query-string tokens can appear in browser history, proxy logs, and diagnostics.
- The browser demo only Base64-encodes text. Base64 is encoding, not encryption.
- Use managed secrets, production-grade Redis and MongoDB configurations, and an explicit origin policy before exposing a deployment publicly.

# This Shit Developed By MOHSEN