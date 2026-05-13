# TODO

Path from local prototype to deployed public API.

## Security (priority)

- [x] Rotate the InfluxDB auth token (the current token was committed to the public repo in `888c14b`)
- [x] Remove `.env` from the repo (`git rm --cached .env`) and add it to `.gitignore`
- [x] Add a `.env.example` with empty values
- [x] Issue a read-only InfluxDB token for production use

## Deploy blockers

- [ ] Make `godotenv.Load` best-effort in `main.go` so a missing `.env` is not fatal in production
- [ ] Add a multi-stage `Dockerfile` with a distroless final image
- [ ] Select a hosting platform (Cloud Run, Fly.io, Railway, or a VM)
- [ ] Deploy and configure a domain

## Production hygiene

- [ ] Replace `http.ListenAndServe` with `http.Server` configured with read, write, and idle timeouts
- [ ] Add graceful shutdown on SIGTERM
- [ ] Initialize the InfluxDB client once in `main` rather than per request
- [ ] Add a `/healthz` endpoint
- [ ] Add CORS headers if a browser client will call the API
- [ ] Update `go.mod` from `go 1.17` to `1.22` or later
- [ ] Refresh dependencies (`go get -u ./... && go mod tidy`)
- [ ] Replace `log.Println` with `log/slog` for structured logging

## Nice to have

- [ ] GitHub Actions workflow running `go test ./...` and `go vet` on pull requests
- [ ] Handler-level tests with a mocked database
- [ ] "Deploying" section in the README
- [ ] Rate limiting or API keys if abuse becomes a concern
