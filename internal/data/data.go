package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/Ribbit-Network/api/internal"
	influxquery "github.com/influxdata/influxdb-client-go/v2/api/query"
)

var (
	errQueryFailed = errors.New("query failed")
	errQueryResult = errors.New("query result error")
)

type Data struct {
	Time time.Time `json:"time"`
	Host string    `json:"host"`

	Altitude    float64 `json:"alt,omitempty"`
	CO2         float64 `json:"co2,omitempty"`
	Humidity    float64 `json:"humidity,omitempty"`
	Latitude    float64 `json:"lat,omitempty"`
	Longitude   float64 `json:"lon,omitempty"`
	Pressure    float64 `json:"baro_pressure,omitempty"`
	Temperature float64 `json:"baro_temperature,omitempty"`
}

type recordIterator interface {
	Next() bool
	Record() *influxquery.FluxRecord
}

var fetchPoints = func(q string) ([]*Data, error) {
	db := internal.NewDB()
	defer db.Close()

	res, err := db.Query(q)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errQueryFailed, err)
	}
	defer res.Close()

	points := collectPoints(res)
	if err := res.Err(); err != nil {
		return nil, fmt.Errorf("%w: %v", errQueryResult, err)
	}
	return getValues(points), nil
}

func Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q, err := NewQuery(r.URL.Query())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	log.Println(q)

	data, err := fetchPoints(q)
	if err != nil {
		msg := "query failed"
		if errors.Is(err, errQueryResult) {
			msg = "query result error"
		}
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	writeJSON(w, data)
}

func collectPoints(res recordIterator) map[string]*Data {
	indexByField := getIndexByField()
	points := make(map[string]*Data)

	for res.Next() {
		rec := res.Record()

		idx, ok := indexByField[rec.Field()]
		if !ok {
			continue
		}

		t := rec.Time()
		host, ok := rec.ValueByKey("host").(string)
		if !ok {
			continue
		}

		key := t.String() + host
		if _, ok := points[key]; !ok {
			points[key] = &Data{Time: t, Host: host}
		}

		v := rec.Value()
		if v == nil {
			continue
		}
		val := reflect.ValueOf(v)
		field := reflect.ValueOf(points[key]).Elem().Field(idx)
		if val.Type().AssignableTo(field.Type()) {
			field.Set(val)
		}
	}

	return points
}

func writeJSON(w http.ResponseWriter, data []*Data) {
	w.Header().Set("Content-Type", "application/json")
	payload := &struct {
		Data []*Data `json:"data"`
	}{Data: data}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Println("json encode error:", err)
	}
}

func getIndexByField() map[string]int {
	indexByField := make(map[string]int)
	t := reflect.TypeOf(Data{})

	for i := 0; i < t.NumField(); i++ {
		tag := strings.Split(t.Field(i).Tag.Get("json"), ",")
		if len(tag) == 2 && tag[1] == "omitempty" {
			indexByField[tag[0]] = i
		}
	}

	return indexByField
}

func getValues(m map[string]*Data) []*Data {
	values := make([]*Data, len(m))
	i := 0
	for _, value := range m {
		values[i] = value
		i++
	}
	return values
}
