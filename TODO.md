# TODO

Path from local prototype to deployed public API.

## Security (priority)

- [x] Rotate the InfluxDB auth token (the current token was committed to the public repo in `888c14b`)
- [x] Remove `.env` from the repo (`git rm --cached .env`) and add it to `.gitignore`
- [x] Add a `.env.example` with empty values
- [x] Issue a read-only InfluxDB token for production use

## Deploy blockers

- [x] Make `godotenv.Load` best-effort in `main.go` so a missing `.env` is not fatal in production
- [x] Add a multi-stage `Dockerfile` with a distroless final image
- [x] Select a hosting platform (Cloud Run, Fly.io, Railway, or a VM)
- [ ] Deploy and configure a domain
- [ ] Set fly.io secrets: `flyctl secrets set INFLUXDB_SERVER_URL=... INFLUXDB_AUTH_TOKEN=... INFLUXDB_ORG=...`
- [ ] Run `flyctl volumes create ribbit_api_data --size 1` then `flyctl deploy`

## Production hygiene

- [x] Replace `http.ListenAndServe` with `http.Server` configured with read, write, and idle timeouts
- [x] Add graceful shutdown on SIGTERM
- [ ] Initialize the InfluxDB client once in `main` rather than per request
- [x] Add a `/healthz` endpoint
- [x] Add CORS headers if a browser client will call the API
- [x] Update `go.mod` from `go 1.17` to `1.22` or later
- [ ] Refresh dependencies (`go get -u ./... && go mod tidy`)
- [ ] Replace `log.Println` with `log/slog` for structured logging

## Nice to have

- [x] GitHub Actions workflow running `go test ./...` and `go vet` on pull requests
- [x] GitHub Actions workflow deploying to fly.io on push to main (requires `FLY_API_TOKEN` secret)
- [ ] Handler-level tests with a mocked database
- [ ] "Deploying" section in the README
- [x] Rate limiting or API keys if abuse becomes a concern (API keys added; rate limiting added)
