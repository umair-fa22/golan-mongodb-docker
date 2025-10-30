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
docker run --rm -e MONGODB_URI="mongodb://host.docker.internal:27017" -p 8080:8080 <yourname>/golan-mongodb-docker:local
```

## Notes

- The workflow's security scan installs `gosec` via `go install` and detects the binary path at runtime to avoid CI path issues.
- Tests that rely on DBs are integration tests and are intentionally separated from unit tests for faster local feedback.
