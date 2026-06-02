# Ribbit Network API

A public API for global CO2 measurements, powered by the [Ribbit Network](https://ribbitnetwork.org) — an open-source network of citizen-operated CO2 sensors.

## 📖 Documentation

Interactive API reference (try requests in the browser):
**[api.ribbitnetwork.org/docs](https://api.ribbitnetwork.org/docs)**

Machine-readable OpenAPI spec:
**[api.ribbitnetwork.org/openapi.yaml](https://api.ribbitnetwork.org/openapi.yaml)**

The spec also lives in this repo at [`internal/docs/openapi.yaml`](internal/docs/openapi.yaml) and is the source of truth — use it to generate client SDKs (`openapi-generator`, `oapi-codegen`, etc.) or to import into Postman / Insomnia / Bruno.

## Quickstart

No key needed to get started — fetch the last day of CO2 readings on the free tier:

```sh
curl "https://api.ribbitnetwork.org/data?start=2024-01-01T00:00:00Z&stop=2024-01-02T00:00:00Z&fields=co2,lat,lon&interval=1h"
```

An [API key](#api-keys) is optional and unlocks much higher [rate limits](#rate-limits) — pass it as a header:

```sh
curl -H "Authorization: Bearer $RIBBIT_API_KEY" \
  "https://api.ribbitnetwork.org/data?start=2024-01-01T00:00:00Z&stop=2024-01-02T00:00:00Z&fields=co2,lat,lon&interval=1h"
```

Endpoints at a glance:

| Endpoint        | Auth      | Description |
|-----------------|-----------|-------------|
| `GET /`         | —         | Health banner (`🐸`) |
| `GET /healthz`  | —         | Liveness check (`ok`) |
| `GET /docs`     | —         | Interactive API reference |
| `GET /data`     | optional  | Sensor measurements over a time range |
| `GET /sensors`  | optional  | List of known sensor IDs |

See **[/docs](https://api.ribbitnetwork.org/docs)** for full parameter, response, and error documentation.

## Rate limits

| Tier            | Limited by | Sustained rate     | Burst |
|-----------------|------------|--------------------|-------|
| Free (no key)   | client IP  | 1 request / minute | 5     |
| Keyed           | API key    | 1 request / second | 60    |

The free tier is sized for polling a single sensor about once a minute. For more sensors or faster polling, [get a key](#api-keys). Exceeding the limit returns `429 Too Many Requests` with a `Retry-After` header.

## Running locally

**Prerequisites:** [Go](https://go.dev/doc/install) 1.25+

1. Clone the repo:
   ```sh
   git clone https://github.com/Ribbit-Network/api && cd api
   ```

2. Copy the example env file and fill in your InfluxDB credentials:
   ```sh
   cp .env.example .env
   ```

3. Run:
   ```sh
   go run .
   ```

The API will be available at `http://localhost:8080`, and the interactive docs at `http://localhost:8080/docs`.

### Previewing just the docs

If you only want to render the OpenAPI page (no InfluxDB or API-key store needed), run:

```sh
go run . docs
```

This serves the embedded spec and Scalar reference at `http://localhost:8080`. Handy when iterating on [`internal/docs/openapi.yaml`](internal/docs/openapi.yaml).

### Environment variables

| Variable              | Description |
|-----------------------|-------------|
| `PORT`                | Port to listen on (e.g. `8080`) |
| `INFLUXDB_SERVER_URL` | InfluxDB Cloud instance URL |
| `INFLUXDB_AUTH_TOKEN` | InfluxDB API token (use a read-only token in production) |
| `INFLUXDB_ORG`        | InfluxDB organization name or email |
| `INFLUXDB_BUCKET`     | InfluxDB bucket name (`frog_fleet`) |
| `API_KEY_DB_PATH`     | Path to the SQLite file holding hashed API keys |
| `SENSORS_CACHE_TTL`   | How long the `/sensors` list is cached (Go duration, e.g. `10m`; default `5m`) |

## API keys

`/data` and `/sensors` are open to anonymous callers on the free tier; an API key is optional and raises the [rate limit](#rate-limits) to the keyed tier. A key that is sent but invalid or revoked is rejected with `401` rather than downgraded to the free tier. Keys live in a SQLite file at `API_KEY_DB_PATH`; only the SHA-256 of each key is stored.

Key management is built into the API binary as a `keygen` subcommand.

Locally:

```sh
# Issue a new key (the raw key is printed once — record it immediately)
go run . keygen issue --owner you@example.com

# List existing keys
go run . keygen list

# Revoke a key by ID
go run . keygen revoke --id 3
```

In production (on Fly.io), run the same subcommands against the deployed binary over SSH:

```sh
fly ssh console -C "/api keygen issue --owner researcher@uni.edu"
fly ssh console -C "/api keygen issue --owner you@example.com"
fly ssh console -C "/api keygen list"
fly ssh console -C "/api keygen revoke --id 7"
```

Callers pass the key in either header:

```sh
curl -H "Authorization: Bearer rbnt_..." "$API_URL/data?start=2024-01-01T00:00:00Z"
curl -H "X-API-Key: rbnt_..."           "$API_URL/data?start=2024-01-01T00:00:00Z"
```

## Updating the docs

The OpenAPI spec at [`internal/docs/openapi.yaml`](internal/docs/openapi.yaml) is embedded into the binary at build time. When you add or change an endpoint:

1. Edit the spec to match.
2. `go build ./...` to verify it still compiles (the spec is `go:embed`-ed).
3. Visit `/docs` locally to spot-check the rendered output.

## Contributing

Feel free to open an issue or PR! We also have enabled the [Github discussion board](https://github.com/Ribbit-Network/api/discussions) if you prefer that.
