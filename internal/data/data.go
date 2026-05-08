package data

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/Ribbit-Network/api/internal"
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

var csvHeaders = []string{"time", "host", "co2", "lat", "lon", "humidity", "baro_pressure", "baro_temperature", "alt"}

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

	db := internal.NewDB()
	defer db.Close()

	res, err := db.Query(q)
	if err != nil {
		http.Error(w, "query failed", http.StatusInternalServerError)
		return
	}
	defer res.Close()

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
	if res.Err() != nil {
		http.Error(w, "query result error", http.StatusInternalServerError)
		return
	}

	data := getValues(points)

	wantsCSV := r.URL.Query().Get("format") == "csv" ||
		strings.Contains(r.Header.Get("Accept"), "text/csv")

	if wantsCSV {
		writeCSV(w, data)
	} else {
		writeJSON(w, data)
	}
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

func writeCSV(w http.ResponseWriter, data []*Data) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=ribbit-data.csv")

	cw := csv.NewWriter(w)
	_ = cw.Write(csvHeaders)

	for _, d := range data {
		_ = cw.Write([]string{
			d.Time.UTC().Format(time.RFC3339),
			d.Host,
			fmt.Sprintf("%g", d.CO2),
			fmt.Sprintf("%g", d.Latitude),
			fmt.Sprintf("%g", d.Longitude),
			fmt.Sprintf("%g", d.Humidity),
			fmt.Sprintf("%g", d.Pressure),
			fmt.Sprintf("%g", d.Temperature),
			fmt.Sprintf("%g", d.Altitude),
		})
	}

	cw.Flush()
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
