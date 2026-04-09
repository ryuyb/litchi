# ---- Stage 1: Build frontend ----
FROM node:22-alpine AS frontend-builder

RUN corepack enable && corepack prepare pnpm@latest --activate

WORKDIR /app/web

COPY web/package.json web/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY web/ ./
RUN pnpm build

# ---- Stage 2: Build backend ----
FROM golang:1.26-alpine AS backend-builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Generate Swagger docs
RUN go install github.com/swaggo/swag/v2/cmd/swag@latest && \
    swag init --v3.1 -g cmd/litchi/server.go -d . -o ./docs/api \
    --parseInternal --parseDependencyLevel 3 --outputTypes go,json,yaml --propertyStrategy camelcase

# Copy frontend build output into static embed path
COPY --from=frontend-builder /app/web/dist internal/infrastructure/static/dist

ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_DATE

RUN CGO_ENABLED=0 GOOS=linux go build \
    -tags embed \
    -ldflags "-X main.Version=${VERSION} -X main.GitCommit=${GIT_COMMIT} -X main.BuildDate=${BUILD_DATE}" \
    -o /bin/litchi ./cmd/litchi

# ---- Stage 3: Runtime ----
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata git

WORKDIR /app

COPY --from=backend-builder /bin/litchi /app/litchi
COPY config/ /app/config/

EXPOSE 8080

ENTRYPOINT ["/app/litchi"]
CMD ["server"]
