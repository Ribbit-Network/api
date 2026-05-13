# Ribbit Network API

A public API for global CO2 measurements, powered by the [Ribbit Network](https://ribbitnetwork.org) — an open-source network of citizen-operated CO2 sensors.

## Endpoints

### `GET /`

Health check. Returns `🐸`.

---

### `GET /data`

Returns CO2, temperature, humidity, and location measurements from the sensor network for a given time range.

#### Query parameters

| Parameter  | Required | Description |
|------------|----------|-------------|
| `start`    | yes      | Start of time range (RFC 3339, e.g. `2024-01-01T00:00:00Z`) |
| `stop`     | no       | End of time range (RFC 3339). Omit to query through the present. |
| `hosts`    | no       | Comma-separated list of sensor IDs to filter by |
| `fields`   | no       | Comma-separated list of fields to return. Available fields: `co2`, `lat`, `lon`, `humidity`, `baro_pressure`, `baro_temperature`, `alt`. Omit to return all fields. |
| `interval` | no       | Aggregate readings into windows of this duration (e.g. `5m`, `1h`). Uses mean aggregation. Omit for raw data. |
| `format`   | no       | Set to `csv` to receive a CSV response instead of JSON. |

You can also request CSV by sending `Accept: text/csv`.

#### JSON response (default)

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

#### CSV response

```
GET /data?start=2024-01-01T00:00:00Z&fields=co2,lat,lon&format=csv
```

```
time,host,co2,lat,lon,humidity,baro_pressure,baro_temperature,alt
2024-01-01T00:00:00Z,a3f2...,412.5,37.77,-122.41,,,, 
...
```

Or with a header:

```sh
curl -H "Accept: text/csv" "https://<host>/data?start=2024-01-01T00:00:00Z"
```

## Running locally

**Prerequisites:** [Go](https://go.dev/doc/install) 1.17+

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

## Contributing

Feel free to open an issue or PR! We also have enabled the Github discussion board if you prefer that.
