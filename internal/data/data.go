package data

import (
	"encoding/json"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/Ribbit-Network/api/internal"
)

type Wrapper struct {
	Data []*Data `json:"data"`
}

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

func Handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		return
	}

	q, err := NewQuery(r.URL.Query())
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	log.Println(q)

	db := internal.NewDB()
	defer db.Close()

	res, err := db.Query(q)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer res.Close()

	t := reflect.TypeOf(Data{})
	fieldIndices := make(map[string]int)

	for i := 0; i < t.NumField(); i++ {
		tag := strings.Split(t.Field(i).Tag.Get("json"), ",")
		if len(tag) == 2 && tag[1] == "omitempty" {
			fieldIndices[tag[0]] = i
		}
	}

	points := make(map[string]*Data)

	for res.Next() {
		rec := res.Record()

		idx, ok := fieldIndices[rec.Field()]
		if !ok {
			continue
		}

		time := rec.Time()
		host := rec.ValueByKey("host").(string)

		key := time.String() + host
		if _, ok := points[key]; !ok {
			points[key] = &Data{Time: time, Host: host}
		}

		val := reflect.ValueOf(rec.Value())
		reflect.ValueOf(points[key]).Elem().Field(idx).Set(val)
	}
	if res.Err() != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	data := &Wrapper{Data: getValues(points)}
	if err := json.NewEncoder(w).Encode(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
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
