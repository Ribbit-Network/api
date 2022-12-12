package data

import (
	"encoding/json"
	"log"
	"net/http"
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

	points := make(map[string]*Data)
	for res.Next() {
		rec := res.Record()
		key := rec.Time().String() + rec.ValueByKey("host").(string)

		if _, ok := points[key]; !ok {
			points[key] = &Data{
				Time: rec.Time(),
				Host: rec.ValueByKey("host").(string),
			}
		}

		switch rec.Field() {
		case "alt":
			points[key].Altitude = rec.Value().(float64)
		case "co2":
			points[key].CO2 = rec.Value().(float64)
		case "humidity":
			points[key].Humidity = rec.Value().(float64)
		case "lat":
			points[key].Latitude = rec.Value().(float64)
		case "lon":
			points[key].Longitude = rec.Value().(float64)
		case "baro_pressure":
			points[key].Pressure = rec.Value().(float64)
		case "baro_temperature":
			points[key].Temperature = rec.Value().(float64)
		}
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
