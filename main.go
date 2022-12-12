package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Ribbit-Network/api/internal/data"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", handle)
	http.HandleFunc("/data", data.Handle)

	addr := fmt.Sprintf(":%s", os.Getenv("PORT"))

	log.Println("API running at http://localhost" + addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handle(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintln(w, "🐸")
}
