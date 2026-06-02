package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "modernc.org/sqlite"

	"github.com/Ribbit-Network/api/internal/auth"
	"github.com/Ribbit-Network/api/internal/data"
	"github.com/Ribbit-Network/api/internal/docs"
	"github.com/Ribbit-Network/api/internal/ratelimit"
	"github.com/Ribbit-Network/api/internal/sensors"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, relying on environment variables")
	}

	if len(os.Args) > 1 && os.Args[1] == "keygen" {
		runKeygen(os.Args[2:])
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "docs" {
		runDocsServer()
		return
	}
	runServer()
}

// runDocsServer serves only the OpenAPI reference. Useful for previewing docs
// changes locally without configuring InfluxDB or the API-key store.
func runDocsServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/docs" {
			http.NotFound(w, r)
			return
		}
		docs.HandleReference(w, r)
	})
	mux.HandleFunc("/openapi.yaml", docs.HandleSpec)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Println("docs preview running at http://localhost:" + port)
	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func runServer() {
	store, err := openKeyStore()
	if err != nil {
		log.Fatal(err)
	}

	// Endpoints are open to anonymous callers but give a higher tier to keyed
	// ones. Optional auth lets unkeyed requests through (rejecting only bad
	// keys); Tiered then limits keyed callers by key and anonymous callers by IP.
	optionalKey := auth.Optional(store)
	// Keyed tier: 1 request/sec with a burst of 60.
	keyed := ratelimit.New(rate.Every(time.Second), 60, 10*time.Minute)
	// Free tier: sized for "poll one sensor once a minute" — 1 req/min sustained
	// with a small burst to absorb startup/retries. ttl >= burst/rate (= 5 min).
	anon := ratelimit.New(rate.Every(time.Minute), 5, 30*time.Minute)
	tier := ratelimit.Tiered(keyed, anon)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/docs", docs.HandleReference)
	mux.HandleFunc("/openapi.yaml", docs.HandleSpec)
	mux.Handle("/data", optionalKey(tier(http.HandlerFunc(data.Handle))))
	mux.Handle("/sensors", optionalKey(tier(http.HandlerFunc(sensors.Handle))))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      corsMiddleware(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Println("API running at http://localhost" + srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}

func openKeyStore() (*auth.Store, error) {
	path := os.Getenv("API_KEY_DB_PATH")
	if path == "" {
		return nil, fmt.Errorf("API_KEY_DB_PATH is required")
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open key db: %w", err)
	}
	return auth.NewStore(db)
}

func handleRoot(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintln(w, "🐸")
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = fmt.Fprintln(w, "ok")
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, X-API-Key, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
