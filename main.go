package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "modernc.org/sqlite"

	"github.com/Ribbit-Network/api/internal/auth"
	"github.com/Ribbit-Network/api/internal/data"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file, relying on environment variables")
	}

	if len(os.Args) > 1 && os.Args[1] == "keygen" {
		runKeygen(os.Args[2:])
		return
	}
	runServer()
}

func runServer() {
	store, err := openKeyStore()
	if err != nil {
		log.Fatal(err)
	}
	requireKey := auth.Require(store)

	http.HandleFunc("/", handle)
	http.Handle("/data", requireKey(http.HandlerFunc(data.Handle)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := fmt.Sprintf(":%s", port)

	log.Println("API running at http://localhost" + addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
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

func handle(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintln(w, "🐸")
}
