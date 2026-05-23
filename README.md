# Ribbit Network API

A public API for global CO2 measurements, powered by the [Ribbit Network](https://ribbitnetwork.org) — an open-source network of citizen-operated CO2 sensors.

## Endpoints

### `GET /`

Health check. Returns `🐸`.

---

### `GET /data`

Returns CO2, temperature, humidity, and location measurements from the sensor network for a given time range.

Requires an API key passed as `Authorization: Bearer <key>` or `X-API-Key: <key>`.

#### Query parameters

| Parameter  | Required | Description |
|------------|----------|-------------|
| `start`    | yes      | Start of time range (RFC 3339, e.g. `2024-01-01T00:00:00Z`) |
| `stop`     | no       | End of time range (RFC 3339). Omit to query through the present. |
| `hosts`    | no       | Comma-separated list of sensor IDs to filter by |
| `fields`   | no       | Comma-separated list of fields to return. Available fields: `co2`, `lat`, `lon`, `humidity`, `baro_pressure`, `baro_temperature`, `alt`. Omit to return all fields. |
| `interval` | no       | Aggregate readings into windows of this duration (e.g. `5m`, `1h`). Uses mean aggregation. Omit for raw data. |

#### JSON response

```
GET /data?start=2024-01-01T00:00:00Z&stop=2024-01-02T00:00:00Z&fields=co2,lat,lon&interval=1h
```

```json
{
    "data": [
        {
            "time": "2024-01-01T00:00:00Z",
            "host": "a3f2...",
            "co2": 412.5,
            "lat": 37.77,
            "lon": -122.41
        },
        ...
    ]
}
```

---

### `GET /sensors`

Returns the list of sensor IDs known to the network (over roughly the last 30 days, per InfluxDB's `schema.tagValues` default).

Requires an API key passed as `Authorization: Bearer <key>` or `X-API-Key: <key>`.

#### JSON response

```
GET /sensors
```

```json
{
    "sensors": ["a3f2...", "b91c...", "..."]
}
```

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
   go run main.go
   ```

The API will be available at `http://localhost:<PORT>`.

## Environment variables

| Variable              | Description |
|-----------------------|-------------|
| `PORT`                | Port to listen on (e.g. `8080`) |
| `INFLUXDB_SERVER_URL` | InfluxDB Cloud instance URL |
| `INFLUXDB_AUTH_TOKEN` | InfluxDB API token (use a read-only token in production) |
| `INFLUXDB_ORG`        | InfluxDB organization name or email |
| `INFLUXDB_BUCKET`     | InfluxDB bucket name (`frog_fleet`) |
| `API_KEY_DB_PATH`     | Path to the SQLite file holding hashed API keys |

## API keys

Access to `/data` requires an API key. Keys live in a SQLite file at `API_KEY_DB_PATH`; only the SHA-256 of each key is stored.

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
fly ssh console -C "/api keygen list"
fly ssh console -C "/api keygen revoke --id 7"
```

Callers pass the key in either header:

```sh
curl -H "Authorization: Bearer rbnt_..." "$API_URL/data?start=2024-01-01T00:00:00Z"
curl -H "X-API-Key: rbnt_..."           "$API_URL/data?start=2024-01-01T00:00:00Z"
```

## Contributing

Feel free to open an issue or PR! We also have enabled the [Github discussion board](https://github.com/Ribbit-Network/api/discussions) if you prefer that.
