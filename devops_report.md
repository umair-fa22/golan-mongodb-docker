# DevOps Report — golan-mongodb-docker

Status: updated (2025-10-30)

This document summarizes the project's DevOps choices, CI/CD pipeline design, runtime architecture, testing approach, security considerations and recommended next steps.

## 1. Architecture overview

- Backend: Go (Gin) application exposing a small CRUD REST API for `items`.
- Database: MongoDB (primary). The repo includes a `docker-compose.yml` that runs both the backend and a `mongo:6.0` service for local development.
- Packaging: multi-stage Dockerfile (build in golang image, produce a tiny runtime image).
- CI: GitHub Actions workflow that builds, scans, tests (with DB services), builds and pushes Docker images.

Runtime flow
- The containerized backend reads `MONGODB_URI` from environment and connects to MongoDB. The compose setup sets `MONGODB_URI` to `mongodb://mongodb:27017/devopsdb` so the backend can connect to the named `mongodb` service on the compose network.

## 2. Key pipeline items

- build-install: `go mod download` + `go build` to ensure compilation.
- security-scan: `gosec` installed via `go install` and executed in the runner (note: detect binary path via `go env GOBIN` or GOPATH/bin).
- test: starts `mongo` and `postgres` as services and runs `go test` (integration tests live in `db_integration_test.go`).
- build-and-push: `buildx` multi-arch Docker build and push to a registry.
- deploy: placeholder in the workflow that runs only on `main` — replace with concrete deployment commands as needed.

CI recommendations
- Split tests by speed: run fast unit tests on PRs and run slow integration tests in a separate job.
- Cache Go modules between runs to speed up build stages.
- Fail the pipeline fast on compile errors (already implemented by build step).

## 3. Observed runtime issue and fix

- Symptom: backend started, connected to MongoDB, then panicked with `html/template: pattern matches no files: 'static/*.html'`, causing the service to crash and the Mongo client to log a disconnected operation.
- Root cause: runtime Docker image originally only copied the binary and not the `static/` folder. `gin.Engine.LoadHTMLGlob("static/*.html")` panicked when templates were missing.
- Fix applied: Dockerfile updated to copy `static/` from the builder into the runtime image. Alternative (recommended): embed `static/` into the Go binary using `embed` so the runtime image only needs the single binary.

## 4. Security & secrets

- Secrets in CI: use GitHub Actions repository secrets for `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` (or registry-specific tokens). Avoid printing secrets in logs.
- Recommendation: use short-lived tokens where supported and rotate periodically.

## 5. Testing

- Integration tests: `db_integration_test.go` exercises MongoDB and Postgres connectivity and basic CRUD. Tests skip when env vars are absent, keeping `go test` usable locally without services.
- Local testing recommendation: use `docker compose up -d mongodb` then run `MONGODB_URI="mongodb://localhost:27017" go test -v ./...`.

## 6. Operational considerations

- Healthchecks: runtime image includes an HTTP healthcheck that probes `/health`. Keep this endpoint lightweight and reliable.
- Startup ordering: Docker Compose `depends_on` only controls start order, not readiness. Consider:
  - Adding a readiness probe in the backend that waits for MongoDB before binding the HTTP listener.
  - Using `wait-for` scripts or a small entrypoint that performs TCP/HTTP checks.
- Data persistence: compose defines a named volume for MongoDB. For production, use managed DB or a replicated MongoDB cluster with backups.

## 7. Monitoring and logging

- Logging: backend uses standard library logging + Gin's logger. For production, forward logs to a centralized system (ELK, Loki, Cloud provider log service).
- Metrics: instrument the app with Prometheus metrics (e.g., request durations, DB latencies) and expose `/metrics`.

## 8. Backup & DR

- For local dev no backups are required. For production:
  - Use `mongodump` or managed provider snapshot features for backups.
  - Define RTO/RPO targets and test restore procedures regularly.

## 9. Scaling and deployment

- For small scale you can run multiple backend replicas behind a load balancer; ensure the MongoDB deployment can handle connections (scale read replicas or use a managed cluster).
- For robust deployment use Kubernetes and a Helm chart or use a managed service (Cloud Run, App Service) and managed MongoDB (Atlas, DocumentDB, CosmosDB, etc.).

## 10. Actionable next steps (priority order)

1. Embed `static/` into the binary using `embed` and remove the need to copy assets into the runtime image. This simplifies images and prevents missing-template panics.
2. Add fast unit tests and separate integration tests in CI (PRs run unit tests; integration tests run in a scheduled or separate job).
3. Add Prometheus metrics and a `/metrics` endpoint.
4. Harden CI: add Trivy or another scanner to the pipeline for image vulnerability scans.
5. If production is intended, add MongoDB authentication and a secure backup strategy; use a managed DB for operational simplicity.

---

If you want, I can implement suggested item #1 (embed static files) and update the Dockerfile so the runtime image is a single binary (no `static/` copies). I can also add the e2e test script that runs after `docker compose up --build` to exercise CRUD endpoints automatically.
# DevOps Report

This report documents the DevOps choices, pipeline design, secret management, testing approach, and lessons learned for the `golan-mongodb-docker` project.

## 1. Technologies used

- Language: Go 1.24
- Web framework: Gin (`github.com/gin-gonic/gin`)
- Database: MongoDB (primary runtime DB) — official Go driver `go.mongodb.org/mongo-driver`
- CI: GitHub Actions
- Container tooling: Docker, Docker Buildx
- Security scanning: gosec
- Test DBs for CI: MongoDB (service container), PostgreSQL (service container) — Postgres used here for demonstration of multi-DB CI testing

## 2. Pipeline design

The pipeline is implemented as a GitHub Actions workflow (`.github/workflows/docker-publish.yml`) and contains five logical stages to meet the assignment requirement.

Mermaid diagram of the pipeline:

```mermaid
flowchart TD
  A[Push/PR] --> B(Build & Install)
  B --> C(Security Scan - gosec)
  C --> D(Test with DB services)
  D --> E(Build Docker Image & Push)
  E --> F(Deploy if branch==main)
```

Stage details
- Build & Install: sets up Go, caches modules, runs `go mod download` and `go build` to validate code compiles.
- Security Scan: installs `gosec` with `go install` and runs it against the codebase to detect common security issues.
- Test (with DB services): launches MongoDB and Postgres as GitHub Actions services, sets environment variables for tests, waits for services to accept connections, and runs `go test -v ./...` which includes integration tests.
- Build Docker Image: uses `docker/setup-buildx-action` and `docker/build-push-action` to build a multi-arch image and push to Docker Hub.
- Deploy (Conditional): runs only when `github.ref == 'refs/heads/main'` and all previous stages pass. Currently a placeholder that prints the pushed image; replace with your deployment commands (SSH, Kubernetes rollout, etc.).

## 3. Secret management strategy

- Use GitHub repository secrets (Settings → Secrets and variables → Actions) to store sensitive values.
- Required secrets for the workflow:
  - `DOCKERHUB_USERNAME` — Docker Hub username or organization
  - `DOCKERHUB_TOKEN` — Docker Hub access token (prefer tokens over passwords)
- Principle: pipeline should not store plaintext credentials in the repository or print secrets to logs. Secrets are only accessed by the `docker/login-action` step and used to tag/push images.
- For deployment targets (e.g., SSH keys or kubeconfig), create separate repository secrets with restrictive names (e.g., `DEPLOY_SSH_KEY`, `KUBE_CONFIG`) and reference them only in the deploy job.

Security best practices
- Use least-privilege tokens for automation (Docker Hub access tokens with limited scope).
- Rotate tokens periodically and revoke if a secret is exposed.

## 4. Testing process

Test types in this repository:

- Unit tests: None added yet — recommended to add fast unit tests for handlers and helpers that run without external services.
- Integration tests: Added in `db_integration_test.go`. These tests:
  - Connect to MongoDB using `MONGODB_URI` and perform a small insert/find/delete on a `ci_items` collection.
  - Connect to Postgres using `POSTGRES_DSN`, create a `ci_test` table, insert a row, read it back, and cleanup.

CI behavior
- The CI `test` job spins up `mongo:6.0` and `postgres:15` as GitHub Actions services.
- Environment variables used in CI:
  - `MONGODB_URI=mongodb://mongo:27017`
  - `POSTGRES_DSN=host=postgres user=postgres password=postgres dbname=testdb sslmode=disable`
- A small wait loop verifies both service ports are reachable before running tests (to avoid flaky failures due to startup time).

Running tests locally
- You can run the integration tests locally by starting matching DB containers and exporting the variables shown in the README.

Test skipping
- Tests skip themselves when the required environment variables are missing. This keeps `go test` usable on machines where DBs are not available.

## 5. Lessons learned

- gosec install path: On GitHub runners `go install` can place binaries in `GOBIN` or in `$(go env GOPATH)/bin`. Trying to call a hard-coded path (e.g., `$GITHUB_WORKSPACE/.gobin/gosec`) caused `No such file or directory`. Fix: detect actual install location at runtime using `go env GOBIN` or fall back to `$(go env GOPATH)/bin` and execute the binary from there.

- Service health checks in workflow YAML: Using `options:` with complex quoted strings caused YAML parsing errors in the workflow file. Safer approach: expose ports and use an explicit wait loop in the job to check service readiness (e.g., `nc -z host port`). If needed, add proper YAML-escaped health options or use held images with known readiness behavior.

- Separate fast and slow tests: Keep a suite of fast unit tests to run on every PR quickly and run slower integration tests in a separate job. This speeds up feedback for code changes.

- Secrets in forked PRs: GitHub masks secrets from workflows triggered by forked PRs. If CI requires secrets for verification, use a gated process or a dedicated CI user.

## Next steps / Recommendations

- Add a suite of unit tests for handlers and small helpers to provide fast PR feedback.
- Implement a real deployment step in the `deploy` job (e.g., publish image to a registry and trigger a Kubernetes deployment or SSH into a host to update containers).
- Add version tagging (semantic-release, Git tags) and branch-aware image tagging (e.g., branch name for feature branches).
- Add vulnerability scanning for Docker images (Trivy) as an extra security stage.

## Appendix — Pipeline excerpt

Key jobs defined in `.github/workflows/docker-publish.yml`:

- `build-install` — compile check and module download
- `security-scan` — runs `gosec` (installed via `go install` and executed from `go env GOBIN` or GOPATH/bin)
- `test` — runs `go test` against services `mongo` and `postgres`
- `build-and-push` — builds multi-arch image and pushes to Docker Hub
- `deploy` — conditional `if: github.ref == 'refs/heads/main'`
