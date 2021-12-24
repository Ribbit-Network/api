package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Ribbit-Network/api/internal/data"
)

const port = 1024

func main() {
	http.HandleFunc("/", handle)
	http.HandleFunc("/data", data.Handle)

	addr := fmt.Sprintf(":%d", port)

	log.Printf("API running at http://localhost:%d\n", port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handle(w http.ResponseWriter, _ *http.Request) {
	_, _ = fmt.Fprintln(w, "🐸")
}
