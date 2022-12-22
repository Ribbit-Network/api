# Ribbit Network API (WIP)

A public API for global CO2 measurements, powered by the Ribbit Network.

## Example

```
GET /data?start=1970-01-01T00:00:00Z&stop=1970-01-01T00:00:10Z
    &hosts=00000000000000000000000000000000,11111111111111111111111111111111
    &fields=co2,lat,lon
    &interval=5m
```

```json
{
    "data": [
        {
            "host": "00000000000000000000000000000000",
            "time": "1970-01-01T00:00:00.000000",
            "co2": 0.0000000000000,
            "lat": 0.00,
            "lon": 0.00
        },
        ...
    ]
}
```

## Running the API

1. Install the latest version of Go: https://go.dev/doc/install
2. Fork and clone the API
3. Run the API locally: `go run main.go`

## Contributing

Feel free to open an issue or PR!
You can also join the developer discord [here](https://discord.com/invite/vq8PkDb2TC).
