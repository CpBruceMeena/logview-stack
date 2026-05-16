# LogView Stack

A fully containerized microstack demonstrating a Go backend, static frontend, PostgreSQL database, Nginx reverse proxy, and a logging pipeline using VictoriaLogs + Promtail.

## Architecture

```
                     ┌──────────────┐
                     │   Browser    │
                     │  :8080       │
                     └──────┬───────┘
                            │
                     ┌──────▼───────┐
                     │    Nginx     │  (reverse proxy)
                     │   :8080 → 80│
                     └──┬──────┬────┘
                        │      │
               ┌────────▼─┐  ┌▼──────────┐
               │ Frontend  │  │  Backend  │  (Go + Gin)
               │(static    │  │ :8080     │
               │ HTML/JS)  │  └──┬───┬────┘
               └───────────┘     │   │
                                 │   │  writes logs to /var/log/backend/app.log
                                 │   │
                         ┌──────▼───▼──────┐
                         │   PostgreSQL    │
                         │   :5432         │
                         └────────────────┘

  Backend logs ──► Promtail ──► VictoriaLogs (:9428)
```

## Services

| Service      | Tech              | Port (host) | Description                       |
|-------------|-------------------|-------------|-----------------------------------|
| `nginx`     | nginx:alpine      | **8080**    | Reverse proxy (single entry point)|
| `frontend`  | nginx:alpine      | —           | Serves static HTML/JS             |
| `backend`   | Go + Gin          | —           | REST API (POST /submit, GET /health)|
| `postgres`  | PostgreSQL 16     | —           | Data store                        |
| `promtail`  | Grafana Promtail 3.6.10 | —           | Log scraper                       |
| `victorialogs` | VictoriaLogs v1.50.0 | **9428**    | Log storage & query               |

## Quick Start

```bash
# Clone / enter the project
cd logview-stack

# Start everything
docker compose up --build
```

The first build downloads Go dependencies and compiles the binary — this may take a minute on the first run.

Once running, open **http://localhost:8080** in your browser.

## API Endpoints

### POST /api/submit

Submit a contact form entry. Accepts `application/x-www-form-urlencoded` or `application/json`.

**Example (curl):**

```bash
curl -X POST http://localhost:8080/api/submit \
  -H "Content-Type: application/json" \
  -d '{"name":"Alice","email":"alice@example.com","message":"Hello from curl!"}'
```

**Or form-encoded:**

```bash
curl -X POST http://localhost:8080/api/submit \
  -d "name=Alice&email=alice@example.com&message=Hello!"
```

**Response (201):**
```json
{"status":"ok","id":1}
```

**Error (400):**
```json
{"error":"all fields are required: name, email, message"}
```

### GET /api/health

Health check endpoint.

```bash
curl http://localhost:8080/api/health
```

**Response (200):**
```json
{"status":"ok","time":"2025-01-01T00:00:00Z"}
```

## Database Connection Details

The backend connects to PostgreSQL using the following environment variables:

- `DB_HOST`: `postgres`
- `DB_PORT`: `5432`
- `DB_USER`: `postgres`
- `DB_PASSWORD`: `password`
- `DB_NAME`: `test`

In this Docker Compose setup, the backend and database share the internal `appnet` network, so the backend uses `postgres` as the hostname.

If you need to inspect the database directly, use the postgres container:

```bash
docker compose exec postgres psql -U postgres -d test
```

> Note: PostgreSQL is not exposed to the host by default in this compose setup.

## Viewing Logs in VictoriaLogs

VictoriaLogs exposes a query interface at **http://localhost:9428**.

> The backend now emits the `_msg` field in JSON logs so VictoriaLogs can ingest them successfully.

### Via the Web UI

Open http://localhost:9428/select/0/vmui/ — you can run LogsQL queries here.

### Example LogsQL Queries

Find all backend submissions:

```
_level:info AND _stream:{job="backend"}
```

Find errors:

```
_level:error AND _stream:{job="backend"}
```

Filter by message content:

```
message: "submission saved"
```

### Via API

```bash
curl -G 'http://localhost:9428/select/0/sql' \
  --data-urlencode "query=SELECT * FROM _stream WHERE job='backend' ORDER BY _time DESC LIMIT 10"
```

## Project Structure

```
logview-stack/
├── backend/
│   ├── Dockerfile
│   ├── go.mod
│   └── main.go
├── frontend/
│   ├── Dockerfile
│   └── index.html
├── nginx/
│   ├── Dockerfile
│   └── nginx.conf
├── postgres-init/
│   └── init.sql
├── promtail/
│   └── config.yaml
├── docker-compose.yml
└── README.md
```

## Stopping

```bash
# Stop containers
docker compose down

# Stop and delete volumes (removes DB data and logs)
docker compose down -v
```

