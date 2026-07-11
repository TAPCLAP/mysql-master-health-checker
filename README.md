# mysql-master-health-checker

HTTPS health-check service for MySQL master nodes. The service exposes a lightweight HTTP endpoint for load balancers and orchestrators and reports **200 OK** only when the node is eligible to receive write traffic.

## Health criteria

The endpoint returns `200 OK` with body `OK` when all conditions are true:

1. MySQL is reachable on the configured host.
2. `@@global.read_only` is not `ON` or `1`.
3. Environment variable `MYSQL_MASTER` is `1` or `true`.

Otherwise the service returns `503 Service Unavailable` with a short reason.

## Architecture

MySQL checks run in a background goroutine on a configurable interval (default: **5s**). HTTP handlers read the latest cached result from memory and only evaluate the cheap `MYSQL_MASTER` environment flag on each request. This avoids hitting MySQL on every probe while still allowing high-frequency HTTP checks from load balancers.

Prometheus metrics are served on a **separate port** (default `127.0.0.1:18849`) over plain HTTP. TLS and binding to `0.0.0.0` are configurable separately for the metrics endpoint.

## Configuration

Every setting is available via environment variable and command-line flag. Environment variables take precedence over flags.

| Variable | Flag | Default | Description |
|----------|------|---------|-------------|
| `LISTEN_ADDR` | `--listen` | `:18848` | HTTPS health-check listen address |
| `TLS_CERT_FILE` | `--tls-cert` | — | Path to TLS certificate (**required**) |
| `TLS_KEY_FILE` | `--tls-key` | — | Path to TLS private key (**required**) |
| `MYSQL_DSN` | `--mysql-dsn` | — | Full MySQL DSN; overrides host/user/password settings |
| `MYSQL_HOST` | `--mysql-host` | `127.0.0.1` | MySQL host when DSN is not set |
| `MYSQL_PORT` | `--mysql-port` | `3306` | MySQL port when DSN is not set |
| `MYSQL_USER` | `--mysql-user` | — | MySQL user when DSN is not set |
| `MYSQL_PASSWORD` | `--mysql-password` | — | MySQL password when DSN is not set |
| `CHECK_INTERVAL` | `--check-interval` | `5s` | Background MySQL polling interval |
| `MYSQL_MASTER` | — | — | Must be `1` or `true` for healthy responses |
| `METRICS_ENABLED` | `--metrics-enabled` | `true` | Enable Prometheus metrics exporter |
| `METRICS_LISTEN_ADDR` | `--metrics-listen` | `127.0.0.1:18849` | Metrics listen address |
| `METRICS_TLS_ENABLED` | `--metrics-tls-enabled` | `false` | Enable TLS for metrics endpoint |
| `METRICS_TLS_CERT_FILE` | `--metrics-tls-cert` | — | TLS certificate for metrics (required when metrics TLS is enabled) |
| `METRICS_TLS_KEY_FILE` | `--metrics-tls-key` | — | TLS private key for metrics (required when metrics TLS is enabled) |

## Endpoints

Health-check (HTTPS):

- `/`
- `/health`
- `/healthz`

Metrics (HTTP by default):

- `/metrics`

## Prometheus metrics

| Metric | Description |
|--------|-------------|
| `mysql_master_health_check_mysql_available` | `1` if MySQL is reachable, `0` otherwise |
| `mysql_master_health_check_mysql_read_only_enabled` | `1` if `read_only` is ON, `0` if OFF, `-1` if unknown |
| `mysql_master_health_check_master_env_enabled` | `1` if `MYSQL_MASTER` is enabled |
| `mysql_master_health_check_unhealthy_reason_info{reason="..."}` | Active failure reason (`mysql_master_env`, `mysql_unavailable`, `mysql_read_only`) |
| `mysql_master_health_check_ok` | `1` if all checks pass (same as HTTP 200), `0` otherwise |
| `mysql_master_health_check_last_mysql_check_timestamp` | Unix timestamp of the last background MySQL check |

## Local development

```bash
make build
export TLS_CERT_FILE=./testdata/tls/server.crt
export TLS_KEY_FILE=./testdata/tls/server.key
export MYSQL_DSN='user:pass@tcp(127.0.0.1:3306)/'
export MYSQL_MASTER=true
./bin/mysql-master-health-checker
```

```bash
curl -k https://127.0.0.1:18848/health
curl http://127.0.0.1:18849/metrics
```

Listen on all interfaces for metrics:

```bash
./bin/mysql-master-health-checker --metrics-listen 0.0.0.0:18849
```

## Make targets

```bash
make test
make test-integration
make lint
make build
make build-release
```

Integration tests start a disposable MySQL container via Docker (testcontainers) and require a running Docker daemon:

```bash
make test-integration
```

## CI

GitHub Actions workflow `.github/workflows/ci.yml` runs lint and tests on every push and pull request. Tag pushes matching `v*` also build statically linked release binaries for `linux/amd64` and `linux/arm64`.

## Container image

```bash
docker build -t mysql-master-health-checker .
```

Pass TLS certificate paths and MySQL settings via environment variables when running the container.
