# Litchi

Turn GitHub Issues into Pull Requests. Litchi runs the development pipeline: clarify requirements, design a solution, break it into tasks, execute them, and open a PR. You review and merge.

## Pipeline

```
Clarification → Design → TaskBreakdown → Execution → PullRequest → Completed
```

You can pause, resume, or roll back at any stage. The agent does the work. You make the calls.

## Architecture

Four layers, DDD-style:

```
Presentation (React) → Application (Services) → Domain (Aggregates) → Infrastructure (GitHub/Git/Agent)
```

The core aggregate is `WorkSession`, which groups an Issue, Clarification, Design, Tasks, and Execution into a single consistency boundary.

## Tech Stack

**Backend** — Go, Fiber v3, GORM + PostgreSQL, Uber Fx, Viper, Zap, golang-migrate

**Frontend** — React, TanStack Start/Router/Query/Table/Form/Store, Tailwind CSS v4, shadcn/ui, Orval (API client generation from OpenAPI)

## Getting Started

Prerequisites: Go 1.26+, Node.js 22+, pnpm, PostgreSQL.

Copy the config and set environment variables:

```bash
cp config/config.yaml config/config.local.yaml

export DB_PASSWORD=your_db_password
export GITHUB_TOKEN=your_github_token
export GITHUB_WEBHOOK_SECRET=your_webhook_secret
```

Run the backend:

```bash
make dev
```

Run the frontend (separate terminal):

```bash
cd web && pnpm install && pnpm dev
```

## Production Build

```bash
make build-embed
```

Builds the frontend, bundles it into the Go binary via `embed.FS`, produces a single static binary.

## Docker

```bash
docker build \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t litchi .

docker run -p 8080:8080 -v ./config:/app/config litchi
```

## Make Targets

| Target | What it does |
|--------|-------------|
| `make dev` | Run backend in dev mode |
| `make build` | Build backend binary (no frontend) |
| `make build-embed` | Build binary with embedded frontend |
| `make frontend-build` | Build frontend |
| `make swagger-gen` | Generate OpenAPI docs |
| `make test` | Run all tests |
| `make test-short` | Run unit tests only |
| `make generate-mocks` | Generate mocks with mockery |

## Documentation

- [`docs/requirements.md`](docs/requirements.md) — Requirements spec
- [`docs/design/architecture.md`](docs/design/architecture.md) — System architecture
- [`docs/design/ddd.md`](docs/design/ddd.md) — Domain model
- [`docs/design/state-machine.md`](docs/design/state-machine.md) — State transitions
- [`docs/tasks/index.md`](docs/tasks/index.md) — Task index and progress
