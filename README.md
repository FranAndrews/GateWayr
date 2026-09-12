# GateWayr

<<<<<<< Updated upstream
=======
A custom-built distributed microservices architecture featuring an API Gateway (Reverse Proxy), round-robin load balancer, internal CRUD APIs, and a middleware stack — all containerised with Docker.

> **Learning project** — built to develop hands-on understanding of network architecture, cloud infrastructure (GCP), containerisation, and enterprise-grade backend patterns.

---

## Architecture

```
┌──────────────┐
│    Client     │
│  (Postman)    │
└──────┬───────┘
       │ HTTP :8080
       ▼
┌──────────────────────────────────────────────┐
│              API GATEWAY                     │
│  ┌────────┐  ┌───────────┐  ┌────────────┐  │
│  │ Logger │→ │ Rate Limit│→ │ API Key    │  │
│  └────────┘  └───────────┘  │ Validator  │  │
│                              └─────┬──────┘  │
│                              ┌─────▼──────┐  │
│                              │ Round-Robin │  │
│                              │ Load Bal.   │  │
│                              └─────┬──────┘  │
│                              ┌─────▼──────┐  │
│                              │  Reverse   │  │
│                              │   Proxy    │  │
│                              └─────┬──────┘  │
└──────────────────────────────────────────────┘
       │          │          │
       ▼          ▼          ▼
  ┌─────────┐ ┌─────────┐ ┌─────────┐
  │backend-1│ │backend-2│ │backend-3│
  │  :9000  │ │  :9000  │ │  :9000  │
  └─────────┘ └─────────┘ └─────────┘
```

## Tech Stack

| Component | Technology |
|-----------|------------|
| API Gateway / Reverse Proxy | Go (`net/http/httputil`) |
| Load Balancer | Go (custom round-robin with `sync/atomic`) |
| Middleware | Go (`log/slog`, `golang.org/x/time/rate`) |
| CRUD APIs | Go 1.22+ enhanced `ServeMux` |
| Database | PostgreSQL 16 (Google Cloud SQL compatible) via `pgx/v5` |
| Database Studio | pgAdmin 4 (`http://localhost:5050`) |
| Containerisation | Docker, Docker Compose |

## Quick Start

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Docker](https://www.docker.com/products/docker-desktop/) & Docker Compose

### Run with Docker Compose

```bash
# Build and start all services (1 gateway + 3 backends)
docker compose up --build

# In another terminal, test round-robin load balancing:
curl http://localhost:8080/health
curl http://localhost:8080/health
curl http://localhost:8080/health
# Each response shows a different "instance" value!
```

### Run locally (without Docker)

```bash
# Terminal 1 — start a backend
PORT=9001 INSTANCE_NAME=local-1 go run ./cmd/backend

# Terminal 2 — start another backend
PORT=9002 INSTANCE_NAME=local-2 go run ./cmd/backend

# Terminal 3 — start the gateway (edit config.yaml backends to localhost:9001, localhost:9002)
go run ./cmd/gateway
```

## API Reference

All endpoints (except `/health`) require the `X-API-Key` header.

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/health` | Health check — returns instance name |
| `POST` | `/api/v1/applications` | Create a new loan application |
| `GET` | `/api/v1/applications` | List all applications |
| `GET` | `/api/v1/applications/{id}` | Get application by ID |
| `PUT` | `/api/v1/applications/{id}` | Update an application |
| `DELETE` | `/api/v1/applications/{id}` | Delete an application |

### Example: Create an application

```bash
curl -X POST http://localhost:8080/api/v1/applications \
  -H "Content-Type: application/json" \
  -H "X-API-Key: gw-dev-key-001" \
  -d '{
    "applicant_name": "John Doe",
    "email": "john@example.com",
    "income_amount": 75000
  }'
```

### Example: Observe round-robin distribution

```bash
# Hit health 6 times — see backend-1, backend-2, backend-3 cycling
for i in $(seq 1 6); do
  echo "Request $i:"
  curl -s http://localhost:8080/health | python -m json.tool
  echo
done
```

### Example: Trigger rate limiting

```bash
# Rapid-fire requests to see 429 responses
for i in $(seq 1 30); do
  echo -n "Request $i: "
  curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health
  echo
done
```

## Project Structure

```
GateWayr/
├── cmd/
│   ├── gateway/main.go         # API Gateway entry point
│   └── backend/main.go         # CRUD API server entry point
├── internal/
│   ├── proxy/proxy.go          # Reverse proxy (httputil.ReverseProxy)
│   ├── loadbalancer/           # Round-robin load balancer + tests
│   ├── middleware/              # Logger, API key auth, rate limiter
│   ├── handler/                # HTTP handlers (CRUD + health)
│   ├── model/                  # Domain models (Application)
│   └── store/                  # Data persistence (interface + in-memory)
├── docker/
│   ├── gateway.Dockerfile
│   └── backend.Dockerfile
├── docker-compose.yml
├── config.yaml
└── go.mod
```

## Middleware Stack

The gateway applies middleware in this order (outermost → innermost):

1. **Logger** — Structured JSON logs for every request (method, path, status, latency, client IP)
2. **Rate Limiter** — Per-IP token-bucket rate limiting (configurable RPS + burst)
3. **API Key Validator** — Validates `X-API-Key` header against allowlist

## Running Tests

```bash
go test ./internal/... -v
```

## Roadmap (Phase 2)

- [ ] Async worker queue (RabbitMQ / Google Pub/Sub) for background document processing
- [ ] Firestore integration for shared data persistence
- [ ] Health-check poller (auto mark backends up/down)
- [ ] GCP deployment (Cloud Run + Artifact Registry)
- [ ] TLS termination

## License

This project is licensed under the GNU General Public License v3.0 — see [LICENSE](LICENSE) for details.
>>>>>>> Stashed changes
