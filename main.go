package main

import (
	"fmt"
	"log"
	"net/http"
)

const port = 1024

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintln(w, "🐸")
	})

	addr := fmt.Sprintf(":%d", port)

	log.Printf("API running at http://localhost:%d\n", port)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
