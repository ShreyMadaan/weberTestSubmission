# AuctionCore Backend Service

AuctionCore is a real-time auction processing backend for live ascending-bid English auctions. It handles atomic bid processing, idempotent bid submission, server-side anti-sniping, synchronous proxy bidding, settlement generation, per-bidder rate limiting, and real-time auction updates through WebSockets.

## Overview

This service rebuilds the core auction engine that processes bids, manages lot lifecycle transitions, and generates settlement invoices for winning bidders. The design prioritizes correctness under concurrency, predictable behavior under retries, and a durable audit trail for every important auction event.

The system is built around:
- PostgreSQL as the source of truth.
- Redis for rate limiting, idempotency coordination, and WebSocket fan-out.
- Background workers for lot closing and settlement.
- An event-sourced auction log for auditability.

## Language Choice Rationale

This implementation uses **Go 1.22**.

Go is a strong fit for this assignment because the system is heavily concurrency-driven and transaction-sensitive. The bid path must handle concurrent requests safely, lock rows correctly, run background workers, and shut down gracefully without leaving partial state behind. Go’s goroutines, channels, and standard library make it well suited for these requirements while keeping the codebase compact and predictable.

Go is also a good match for:
- Low-latency HTTP APIs.
- Database transaction workflows.
- Redis-based background coordination.
- WebSocket fan-out.
- Clean operational behavior under load.

TypeScript would also be acceptable, but Go is easier to keep deterministic for the lock-heavy auction workflow.

## System Architecture

The application is split into the following parts:

- **HTTP API server** for bids, auctions, lots, bidder registration, and settlement reports.
- **PostgreSQL 15** as the source of truth for auction state, bids, idempotency, events, and invoices.
- **Redis 7** for token-bucket rate limiting, idempotency coordination, and pub/sub broadcasting.
- **Lot closer worker** that transitions closing lots to sold or passed.
- **Settlement worker** that captures payments and updates invoice status.
- **WebSocket gateway** for real-time auction updates.

The key design principle is simple: all authoritative state changes happen in PostgreSQL inside transactions, and all real-time notifications are published only after the transaction commits.

## Repository Structure

```text
.
├── cmd/
│   ├── api/
│   ├── lot-closer/
│   └── settlement-worker/
├── db/
│   └── migrations/
├── internal/
│   ├── config/
│   ├── db/
│   ├── handlers/
│   ├── middleware/
│   ├── models/
│   ├── repository/
│   ├── services/
│   ├── websocket/
│   └── workers/
├── mock-services/
│   ├── payment-gateway/
│   └── identity-verification/
├── scripts/
│   └── seed.go
├── docker-compose.yml
└── README.md
```

## Core Features

### Atomic bid processing
Every bid is processed inside a database transaction with a row lock on the lot. This prevents two concurrent bids from both winning the same increment.

### Idempotent bid submission
Each bid request must include an `Idempotency-Key`. Duplicate retries return the same response instead of creating duplicate bids.

### Server-side anti-sniping
If a bid arrives near the end of a lot, the closing time is extended on the server, not in the client.

### Synchronous proxy bidding
Proxy bids are resolved immediately within the same transaction. There is no cron-based delay and no visible lag for other bidders.

### Event-sourced auction state
Each meaningful auction action is written to `auctionevents` so the full sequence of changes can be reconstructed later.

### Incremental settlement
Winning lots create invoices automatically, and a background worker captures payment in a retry-safe way.

### Per-bidder rate limiting
Redis token buckets limit abusive bid traffic per bidder and per lot.

### Real-time updates
WebSocket clients subscribed to an auction receive bid, anti-snipe, and lot-closed events in real time.

## Database Schema

The database includes the following core tables:

- `auctionhouses`
- `auctions`
- `lots`
- `bidders`
- `auctionregistrations`
- `bids`
- `bididempotencykeys`
- `auctionevents`
- `settlementinvoices`
- `lotstatetransitions`

A trigger enforces valid lot status transitions so the application cannot accidentally move a lot into an invalid state.

## Bid Processing Flow

A bid submission follows this sequence:

1. Validate authentication and `Idempotency-Key`.
2. Check rate limits in Redis.
3. Lock or create the idempotency row using `SELECT FOR UPDATE SKIP LOCKED`.
4. Return a cached response if the key already exists and is complete.
5. Lock the lot row with `SELECT FOR UPDATE`.
6. Validate lot state, bidder registration, eligibility, and minimum increment.
7. Insert the bid record.
8. Update the lot’s current bid, winner, and bid count.
9. Extend the lot if anti-sniping applies.
10. Resolve any proxy bids synchronously in the same transaction.
11. Append all state changes to `auctionevents`.
12. Commit the transaction.
13. Publish the resulting event to Redis Pub/Sub for WebSocket fan-out.

This ordering prevents lost updates, double-winning bids, and inconsistent auction state.

## Idempotency Design

Idempotency is handled with a dedicated `bididempotencykeys` table.

Behavior:
- If `Idempotency-Key` is missing, return `400`.
- If another request is already processing the same key, return `409`.
- If a completed response exists, return the original response body and status.
- If the handler fails or panics, the lock must be cleared so the key does not remain stuck.
- Idempotency records expire after one hour.

Important edge cases:
- Network retry after timeout.
- Two concurrent requests with the same key.
- Handler crash while the key is locked.
- Expired idempotency keys that should no longer be reused.

## Anti-Sniping and Proxy Bidding

Anti-sniping is enforced server-side. If a bid is accepted within the configured anti-snipe window before `closingat`, the system extends the closing time by the configured extension and emits an anti-snipe event.

Proxy bidding is resolved synchronously. After a live bid is accepted, the processor checks whether any registered bidder has a proxy maximum that should outbid the current amount. If so, it immediately resolves that proxy as a new auto-increment bid, and repeats until no higher eligible proxy remains.

The proxy resolution loop must always move upward in bid value and must stop when no valid proxy ceiling exceeds the current amount. This prevents infinite loops.

## Settlement Architecture

Settlement is event-driven, not batch-only.

When a lot closer determines that a lot should be sold:
- The lot status is updated in the same transaction.
- A `settlementinvoices` row is created immediately.
- The invoice stores hammer price, buyer premium, total due, currency, due date, and status.

The settlement worker then processes pending invoices by calling the payment gateway mock.

Retry behavior:
- On approval, mark the invoice paid and store the payment reference.
- On decline, retry up to 3 times with 1-hour backoff.
- After retries are exhausted, mark the invoice overdue.

Duplicate capture prevention:
- Before calling the gateway, check whether `paymentref` is already set.
- If it is already set, skip the capture request entirely.

This makes the settlement worker safe to run repeatedly without charging the same invoice twice.

## WebSocket Fan-Out

WebSocket clients subscribe to a specific auction using the auction ID. All bid and lot events for that auction are broadcast to connected clients.

Broadcasted event types:
- `BIDACCEPTED`
- `ANTISNIPE`
- `LOTCLOSED`

The fan-out layer uses Redis Pub/Sub so any API instance can publish events and every WebSocket server instance can forward them to its connected clients.

Heartbeat behavior:
- Send a ping every 10 seconds.
- Close connections that do not answer with a pong within 5 seconds.

If a Redis node is unavailable, real-time fan-out may be degraded, but the auction state remains safe because PostgreSQL is still the system of record.

## External Mock Services

The project includes two required mock services.

### Payment Gateway Mock
A simple HTTP service that simulates payment capture.

Endpoint:
- `POST /capture`

Behavior:
- Returns `200` with `approved` and a payment reference for successful captures.
- Returns `402` with `declined` and a decline reason for failed captures.
- Supports a configurable decline rate for testing retry paths.

### Identity Verification Mock
A simple HTTP service that simulates bidder eligibility checks.

Endpoint:
- `GET /verify?bidderid=...&auctionid=...`

Behavior:
- Returns eligibility status.
- Returns bidder tier such as `basic`, `verified`, or `premium`.
- Returns deposit requirements when applicable.

## API Endpoints

### Lots
- `POST /api/v1/lots/:id/bids`
- `GET /api/v1/lots/:id`
- `GET /api/v1/lots/:id/bids`

### Auctions
- `GET /api/v1/auctions/:id/lots`
- `POST /api/v1/auctions/:id/open`
- `POST /api/v1/auctions/:id/pause`
- `POST /api/v1/auctions/:id/close`
- `GET /api/v1/auctions/:id/settlement`

### Bidders
- `POST /api/v1/bidders/:id/register`

### WebSocket
- `WS /ws/auctions/:id`

## Error Responses

The API uses predictable error responses.

Examples:
- `400` — missing `Idempotency-Key`.
- `409` — outbid, lot closed, or idempotency conflict.
- `422` — bid below minimum increment.
- `429` — rate limit exceeded.

Rate-limit responses include:
- `X-RateLimit-Limit`
- `X-RateLimit-Remaining`
- `X-RateLimit-Reset`
- `Retry-After`

## Rate Limiting

Bid submission is protected by Redis token buckets.

Rules:
- 60 bids per minute global limit per bidder.
- 10 bids per minute per lot.
- Token replenishment is continuous, not fixed-window.
- Lua scripts are used for atomic check-and-decrement.

If either bucket is empty, the request is rejected with `429 Too Many Requests`.

## Increment Rules

Minimum bid increments are based on the current price:

- `100` to `499` → `5`
- `500` to `999` → `10`
- `1,000` to `4,999` → `25`
- `5,000` to `19,999` → `250`
- `20,000` to `99,999` → `1,000`
- `100,000+` → `5,000`

These values are applied to determine the minimum valid next bid.

## Settlement Report

The settlement report endpoint returns an aggregate view of all lots in an auction, including:
- lot title,
- sold price,
- invoice status,
- payment reference,
- total realized value.

Access is restricted to the auction house operator.

## Prometheus Metrics

The service exposes Prometheus metrics for observability.

Required metrics include:
- `bids_total{status=...}` — count of bids by final outcome.
- `bid_processing_duration_seconds` — bid processing latency.
- `antisnipe_triggers_total` — number of anti-sniping extensions triggered.
- `settlement_invoices_total{status=...}` — invoice counts by status.

Optional but useful metrics:
- rate-limit rejections,
- idempotency cache hits,
- proxy resolution count,
- lot closer iterations,
- payment success/failure counts.

## Testing Strategy

The project includes tests for the most important concurrency and correctness paths:

- concurrent bid submissions on the same lot,
- idempotent request handling,
- idempotency lock release on failure,
- anti-sniping extension behavior,
- synchronous proxy resolution,
- settlement retry and duplicate-capture prevention,
- rate limiting behavior.

The critical bid processor and related flows are targeted for at least 75% coverage.

## Running Locally

### Prerequisites
- Go 1.22
- Docker and Docker Compose
- PostgreSQL 15
- Redis 7

### Start dependencies
```bash
docker compose up -d
```

### Run migrations
```bash
make migrate
```

### Seed the database
```bash
make seed
```

### Start the API server
```bash
make run-api
```

### Start background workers
```bash
make run-workers
```

### Example request: place a bid
```bash
curl -X POST http://localhost:8080/api/v1/lots/<lot-id>/bids \
  -H 'Authorization: Bearer <token>' \
  -H 'Idempotency-Key: abc-123' \
  -H 'Content-Type: application/json' \
  -d '{
    "amountcents": 125000,
    "bidtype": "live"
  }'
```

### Example request: open an auction
```bash
curl -X POST http://localhost:8080/api/v1/auctions/<auction-id>/open \
  -H 'Authorization: Bearer <operator-token>'
```

### Example request: fetch a lot snapshot
```bash
curl http://localhost:8080/api/v1/lots/<lot-id>
```

## Graceful Shutdown

The service handles `SIGTERM` by:
- stopping new request intake,
- waiting for in-flight bids to complete for up to 10 seconds,
- finishing the current lot-closer iteration,
- closing database and Redis connections cleanly.

This prevents partial updates and helps the service stop safely in containerized environments.

## Design Goals

This system is designed around four goals:

- **Correctness** under concurrency.
- **Auditability** through event logging.
- **Resilience** through idempotency and retries.
- **Operational safety** through background workers, rate limiting, and graceful shutdown.

Those goals match the needs of a real auction platform where trust and timing are critical.

## License

This repository is intended for assessment and interview submission purposes only.
