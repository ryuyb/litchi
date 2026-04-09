# Setup Guide

This guide walks through setting up Litchi from scratch: installing prerequisites, creating a GitHub App, configuring the application, and running it locally.

## Prerequisites

- Go 1.26+
- Node.js 22+
- pnpm
- PostgreSQL 15+
- Git

## Step 1: Create a GitHub App

Litchi supports two authentication methods:

- **PAT** (Personal Access Token): simpler, good for personal projects or getting started.
- **GitHub App**: recommended for team or multi-repo use. Supports per-installation tokens and webhook events.

### Option A: Personal Access Token (Quick Start)

1. Go to **Settings** > **Developer settings** > **Personal access tokens** > **Fine-grained tokens**.
2. Generate a new token with these permissions:
   - Repository permissions: **Issues** (read/write), **Pull requests** (read/write), **Contents** (read/write), **Metadata** (read)
3. Save the token. You will set it as `GITHUB_TOKEN`.

### Option B: GitHub App (Recommended)

1. Go to **Settings** > **Developer settings** > **GitHub Apps** > **New GitHub App**.

2. Fill in the form:
   - **GitHub App name**: `Litchi` (or your preferred name)
   - **Homepage URL**: your deployment URL or `https://github.com/your-org/litchi`
   - **Webhook URL**: `https://your-domain.com/api/v1/webhooks/github`
   - **Webhook secret**: generate a random string and save it. You will set it as `GITHUB_WEBHOOK_SECRET`.

3. Set repository permissions:

   | Permission | Access | Why |
   |-----------|--------|-----|
   | Issues | Read & write | Read issues, post clarification comments |
   | Pull requests | Read & write | Create PRs, read reviews |
   | Contents | Read & write | Clone repos, push branches |
   | Metadata | Read | Required (default) |
   | Commit statuses | Read & write | Report build/test results |

4. Subscribe to webhook events:

   | Event | Why |
   |-------|-----|
   | Issues | Trigger sessions when issues are created or labeled |
   | Issue comment | Receive answers to clarification questions |
   | Pull request | Track PR state changes |
   | Pull request review | Receive review approvals or change requests |
   | Push | Monitor branch updates |
   | Installation | Handle app install/uninstall lifecycle |

5. Under "Where can this GitHub App be installed?", choose **Any account** (or restrict to specific orgs).

6. Click **Create GitHub App**.

7. After creation:
   - Note the **App ID** (visible on the app settings page). You will set it as `GITHUB_APP_ID`.
   - Under "Private keys", generate a new private key. Download the `.pem` file. You will set its path as `GITHUB_PRIVATE_KEY_PATH`.

8. Install the app on your repositories:
   - Go to the app settings page, click **Install App** in the sidebar.
   - Select the account and repositories where Litchi should operate.

## Step 2: Configure the Application

Configuration is layered. Files are loaded in this order, with later entries overriding earlier ones:

1. `config/config.yaml` — base configuration (checked into git)
2. `config/config.{LITCHI_ENV}.yaml` — environment-specific overrides (`dev`, `uat`, `prod`)
3. `config/config.local.yaml` — local developer overrides (git-ignored)

You can also override any value with environment variables prefixed by `LITCHI_`. For example, `LITCHI_SERVER_PORT=9090` overrides `server.port`.

### 2.1 Create Your Local Config

```bash
cp config/config.yaml config/config.local.yaml
```

### 2.2 Set Required Environment Variables

```bash
# Database password (required)
export DB_PASSWORD=your_db_password

# GitHub authentication — choose one method:

# Method A: PAT
export GITHUB_TOKEN=ghp_your_personal_access_token

# Method B: GitHub App (takes precedence if both are set)
export GITHUB_APP_ID=123456
export GITHUB_PRIVATE_KEY_PATH=/path/to/your-app.private-key.pem

# Webhook secret (required for webhook signature verification)
export GITHUB_WEBHOOK_SECRET=your_webhook_secret
```

### 2.3 Adjust config.local.yaml

Edit `config/config.local.yaml` to match your local setup:

```yaml
server:
  port: 8080
  mode: "debug"
  enable_swagger: true

database:
  host: "localhost"
  port: 5432
  name: "litchi"
  user: "postgres"
  password: "postgres"
  sslmode: "disable"
  auto_migrate: true

github:
  token: "dummy"            # Overridden by GITHUB_TOKEN env var
  webhook_secret: "dummy"   # Overridden by GITHUB_WEBHOOK_SECRET env var

git:
  worktree_base_path: "/tmp/litchi/worktrees"
  worktree_auto_clean: true
  default_base_branch: "main"

logging:
  level: "debug"
  format: "console"
```

### 2.4 Environment Detection

Litchi detects the environment from the first set variable among `LITCHI_ENV`, `GO_ENV`, `ENV`, defaulting to `dev`.

```bash
# Run in UAT mode
export LITCHI_ENV=uat
```

This loads `config/config.uat.yaml` on top of the base config.

## Step 3: Set Up the Database

Create a PostgreSQL database:

```bash
createdb litchi
```

With `auto_migrate: true` in your local config, tables are created on startup. For production, set `auto_migrate: false` and use migrations explicitly.

## Step 4: Run the Application

### Backend

```bash
make dev
```

The server starts on `http://localhost:8080`. Swagger UI is available at `http://localhost:8080/swagger`.

### Frontend (Separate Terminal)

```bash
cd web
pnpm install
pnpm dev
```

## Step 5: Verify the Setup

1. **Backend health**: open `http://localhost:8080/api/v1/health`.
2. **Swagger UI**: open `http://localhost:8080/swagger`.
3. **Webhook health**: open `http://localhost:8080/api/v1/webhooks/health`.
4. **GitHub connection**: create a test issue in a repo where the app is installed, then check the logs for incoming webhook events.

## Production Build

Build a single static binary with the frontend embedded:

```bash
make build-embed
```

Or use Docker:

```bash
docker build \
  --build-arg VERSION=$(git describe --tags --always) \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t litchi .

docker run -p 8080:8080 \
  -v ./config:/app/config \
  -e DB_PASSWORD=your_password \
  -e GITHUB_TOKEN=your_token \
  -e GITHUB_WEBHOOK_SECRET=your_secret \
  litchi
```

## Configuration Reference

The full base config with all available options and comments is in [`config/config.yaml`](../config/config.yaml). Environment-specific overrides live in `config/config.{dev,uat,prod}.yaml`.

Key sections:

| Section | What it controls |
|---------|-----------------|
| `server` | Host, port, mode (debug/release), Swagger toggle |
| `database` | PostgreSQL connection, pool sizing, auto-migrate |
| `github` | Token or App credentials, webhook secret |
| `git` | Worktree paths, branch naming, commit signing |
| `webhook` | Idempotency, TTL, auto-cleanup |
| `agent` | Agent type, concurrency, retry limits, approval timeout |
| `clarity` | Thresholds for auto-proceed vs. forced clarification |
| `complexity` | Design confirmation thresholds |
| `audit` | Audit logging toggle, retention, sensitive operation tracking |
| `failure` | Retry backoff, rate limiting, per-stage timeouts, queue limits |
| `logging` | Log level, output targets (console/file), rotation |
| `middleware` | CORS, rate limiting, CSRF, compression |
| `redis` | Redis toggle (optional, disabled by default in dev) |
