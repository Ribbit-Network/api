package sensors

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Ribbit-Network/api/internal"
	influxquery "github.com/influxdata/influxdb-client-go/v2/api/query"
)

var (
	errQueryFailed = errors.New("query failed")
	errQueryResult = errors.New("query result error")
)

type recordIterator interface {
	Next() bool
	Record() *influxquery.FluxRecord
}

var fetchSensors = func() ([]string, error) {
	db := internal.NewDB()
	defer db.Close()

	res, err := db.Query(buildQuery(os.Getenv("INFLUXDB_BUCKET")))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errQueryFailed, err)
	}
	defer res.Close()

	ids := collectIDs(res)
	if err := res.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", errQueryResult, err)
	}
	return ids, nil
}

func Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ids, err := fetchSensors()
	if err != nil {
		msg := "query failed"
		if errors.Is(err, errQueryResult) {
			msg = "query result error"
		}
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	writeJSON(w, ids)
}

func buildQuery(bucket string) string {
	return fmt.Sprintf(`import "influxdata/influxdb/schema"
schema.tagValues(bucket: "%s", tag: "host")`, bucket)
}

func collectIDs(res recordIterator) []string {
	ids := []string{}
	for res.Next() {
		if v, ok := res.Record().Value().(string); ok {
			ids = append(ids, v)
		}
	}
	return ids
}

func writeJSON(w http.ResponseWriter, ids []string) {
	w.Header().Set("Content-Type", "application/json")
	if ids == nil {
		ids = []string{}
	}
	payload := struct {
		Sensors []string `json:"sensors"`
	}{Sensors: ids}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Println("json encode error:", err)
	}
}
