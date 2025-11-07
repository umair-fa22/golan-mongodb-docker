## Golan MongoDB Docker

A small Go REST API (Gin) that persists items in MongoDB. This repository includes:

- A Go backend with CRUD HTTP endpoints.
- A multi-stage Dockerfile for small runtime images.
- `docker-compose.yml` for local development (backend + MongoDB).
- Integration tests and a GitHub Actions pipeline that builds, scans, tests, and publishes the image.

---

## Quick start (recommended)

Prerequisites
- Docker & Docker Compose (for running the app and MongoDB together)
- Go (only required if you plan to run locally outside Docker)

Start with Docker Compose (builds the backend image and starts MongoDB):

```bash
# from the repository root
docker compose up --build
```

After startup the backend will be reachable at:

http://localhost:8080

Open the web UI at `/` or use the API under `/api` (see endpoints below).

To stop and remove containers and the anonymous network (persistent data is kept in a named volume):

```bash
docker compose down
```

---

## Environment variables

The app reads configuration from environment variables. When using `docker-compose.yml` these are already provided. Key variables:

- `MONGODB_URI` (required) — MongoDB connection string (e.g. `mongodb://mongodb:27017/devopsdb`). The app also accepts the legacy `MONGO_URI` if set.
- `PORT` — HTTP port the backend listens on (defaults to `8080`).

If you run the binary locally, export `MONGODB_URI` first. Example:

```bash
export MONGODB_URI="mongodb://localhost:27017"
export PORT=8080
go run ./...
```

---

## API Endpoints

All endpoints use JSON for request and response bodies.

- GET  /api/items           — list all items
- GET  /api/items/:id       — get single item by MongoDB ObjectID
- POST /api/items           — create an item (body: name, unitPrice, quantity)
- PUT  /api/items/:id       — update an item
- DELETE /api/items/:id     — delete an item

Health endpoint:
- GET /health — returns {"status":"ok"} for container healthchecks and probes.

Example: create an item

```bash
curl -s -X POST http://localhost:8080/api/items \
  -H "Content-Type: application/json" \
  -d '{"name":"widget","unitPrice":9.99,"quantity":10}'
```

---

## Running tests

There are integration tests in `db_integration_test.go` that exercise MongoDB (and Postgres for demonstration). Tests look for `MONGODB_URI` and `POSTGRES_DSN` environment variables and will skip if those are not set.

Quick local flow using Docker for the MongoDB dependency:

```bash
# start only the mongo service from compose
docker compose up -d mongodb

export MONGODB_URI="mongodb://localhost:27017"
go test -v ./...

# stop mongodb when done
docker compose down
```

---

## Troubleshooting (common issues)

- Panic on startup: `html/template: pattern matches no files: 'static/*.html'`
  - Cause: runtime image didn't include the `static/` directory. Fixed by copying `static/` into the runtime image in the Dockerfile. If you still see this, confirm your image was rebuilt with `docker compose up --build`.
- MongoDB connection errors
  - Ensure `MONGODB_URI` is set and reachable from where the app runs. When using Compose the service host is `mongodb` (e.g. `mongodb://mongodb:27017/devopsdb`).
- Tests skipped
  - Integration tests skip when required environment variables aren't present. Export `MONGODB_URI` to run them locally.

---

## Implementation notes & recommendations

- Static assets: For single-binary images consider using Go's `embed` package to embed the `static/` files into the binary. This removes the need to copy assets into the runtime image and avoids missing-file panics.
- Healthchecks & startup ordering: Docker Compose `depends_on` controls start order but not readiness. Use a small wait-for script or make the backend poll MongoDB until it becomes healthy to avoid transient errors at startup.
- Production readiness: Current compose setup is intended for local dev. For production, enable MongoDB authentication, backups, and use a proper orchestrator (Kubernetes) or managed MongoDB service.

---

If you want, I can:

- Embed static files into the binary and update the Dockerfile so runtime images are single-file.
- Add a small e2e script that exercises the CRUD endpoints after starting `docker compose up --build`.
- Add a readiness wait in the backend so it delays accepting requests until MongoDB is reachable.

Happy to make any of those changes — tell me which one you prefer.
# Golan MongoDB Docker

Simple Go REST API using Gin and MongoDB, packaged with Docker and CI via GitHub Actions.

This repository demonstrates:

- A Go web service using Gin and the official MongoDB Go driver.
- A multi-stage Dockerfile (builder + minimal runtime).
- A GitHub Actions pipeline with stages: Build & Install, Security Scan, Test (with DB services), Build Docker Image, and Deploy (conditional).

## Quick start (development)

Prerequisites
- Go 1.24+ (for local dev)

- Docker (optional, for running the image)

Run locally (connect a MongoDB instance and set MONGODB_URI):

PowerShell example:

```powershell
# Set MongoDB URI (for local Mongo or a container)
$env:MONGODB_URI = "mongodb://localhost:27017"
$env:PORT = "8080"
# Run
go run ./...
```

Open http://localhost:8080 in the browser.

API endpoints (JSON)
- GET /api/items
- GET /api/items/:id
- POST /api/items
- PUT /api/items/:id
- DELETE /api/items/:id

## Run tests

There are integration tests that require MongoDB and Postgres. Locally you can run them by setting environment variables to point at running DBs.

Example (PowerShell) when running DB containers locally:

```powershell
docker run -d --name mongo -p 27017:27017 mongo:6.0
docker run -d --name pg -e POSTGRES_PASSWORD=postgres -e POSTGRES_USER=postgres -e POSTGRES_DB=testdb -p 5432:5432 postgres:15

$env:MONGODB_URI = "mongodb://localhost:27017"
$env:POSTGRES_DSN = "host=localhost user=postgres password=postgres dbname=testdb sslmode=disable"

go test -v ./...
```

If the env variables are not set, the integration tests will be skipped.

## CI / GitHub Actions

The workflow file is at `.github/workflows/docker-publish.yml` and contains these stages:

1. Build & Install — go mod download, go build
2. Security Scan — run `gosec` on the code
3. Test (with DB services) — spins up MongoDB and Postgres service containers and runs `go test`
4. Build Docker Image — uses `docker/build-push-action` and `buildx` to build multi-platform images and push to Docker Hub
5. Deploy (Conditional) — placeholder step that runs only on `main` after successful build & push

Secrets required (set in GitHub repository Settings → Secrets → Actions):
- `DOCKERHUB_USERNAME` — Docker Hub username or org
- `DOCKERHUB_TOKEN` — Docker Hub access token (recommended)

## Docker

Build locally:

```powershell
docker build -t <yourname>/golan-mongodb-docker:local .
```

Run the image (make sure MongoDB is reachable from inside the container or set MONGODB_URI to a network-accessible Mongo):

```powershell
docker run --rm -e MONGODB_URI="mongodb://host.docker.internal:27017" -p 8080:8080 umair-fa22/golan-mongodb-docker:local
```

Docker Compose (recommended for local dev)

```bash
# Build and start the app + MongoDB defined in docker-compose.yml
docker compose up --build

# The backend will be reachable at http://localhost:8080
# The compose file provides MONGODB_URI and exposes Mongo on 27017

# To stop and remove containers + volumes (data persisted in named volume):
docker compose down
```

## Notes

- The workflow's security scan installs `gosec` via `go install` and detects the binary path at runtime to avoid CI path issues.
- Tests that rely on DBs are integration tests and are intentionally separated from unit tests for faster local feedback.
