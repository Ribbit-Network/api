package data

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	influxquery "github.com/influxdata/influxdb-client-go/v2/api/query"
	"github.com/stretchr/testify/require"
)

func TestGetValues(t *testing.T) {
	a := time.Unix(0, 0)
	b := time.Unix(1, 0)

	m := map[string]*Data{"a": {Time: a}, "b": {Time: b}}
	v := getValues(m)

	require.Contains(t, v, &Data{Time: a})
	require.Contains(t, v, &Data{Time: b})
}

// withFetchPoints swaps the package-level fetchPoints hook for the duration
// of a test and restores it afterward.
func withFetchPoints(t *testing.T, fn func(string) ([]*Data, error)) {
	t.Helper()
	orig := fetchPoints
	fetchPoints = fn
	t.Cleanup(func() { fetchPoints = orig })
}

func sampleData() []*Data {
	return []*Data{{
		Time:        time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC),
		Host:        "frog-01",
		CO2:         421.5,
		Latitude:    47.6,
		Longitude:   -122.3,
		Humidity:    55.2,
		Pressure:    1013.1,
		Temperature: 21.0,
		Altitude:    12.0,
	}}
}

func TestHandle_MethodNotAllowed(t *testing.T) {
	withFetchPoints(t, func(string) ([]*Data, error) {
		t.Fatal("fetchPoints should not be called for non-GET")
		return nil, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/data?start=2024-01-01T00:00:00Z", nil)
	rec := httptest.NewRecorder()

	Handle(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestHandle_BadStart_Returns400WithBody(t *testing.T) {
	withFetchPoints(t, func(string) ([]*Data, error) {
		t.Fatal("fetchPoints should not be called when start is invalid")
		return nil, nil
	})

	cases := []struct {
		name string
		url  string
	}{
		{"missing", "/data"},
		{"unparseable", "/data?start=not-a-timestamp"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			rec := httptest.NewRecorder()

			Handle(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Code)
			require.NotEmpty(t, strings.TrimSpace(rec.Body.String()), "400 response should include an error body")
		})
	}
}

func TestHandle_DefaultReturnsJSON(t *testing.T) {
	withFetchPoints(t, func(string) ([]*Data, error) {
		return sampleData(), nil
	})

	req := httptest.NewRequest(http.MethodGet, "/data?start=2024-01-01T00:00:00Z", nil)
	rec := httptest.NewRecorder()

	Handle(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var payload struct {
		Data []*Data `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Len(t, payload.Data, 1)
	require.Equal(t, "frog-01", payload.Data[0].Host)
}

func TestHandle_FormatCSV_ReturnsCSV(t *testing.T) {
	withFetchPoints(t, func(string) ([]*Data, error) {
		return sampleData(), nil
	})

	req := httptest.NewRequest(http.MethodGet, "/data?start=2024-01-01T00:00:00Z&format=csv", nil)
	rec := httptest.NewRecorder()

	Handle(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/csv", rec.Header().Get("Content-Type"))

	rows, err := csv.NewReader(bytes.NewReader(rec.Body.Bytes())).ReadAll()
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 2, "expected header row plus at least one data row")
	require.Equal(t, csvHeaders, rows[0])
	require.Equal(t, "frog-01", rows[1][1])
}

func TestHandle_AcceptCSV_ReturnsCSV(t *testing.T) {
	withFetchPoints(t, func(string) ([]*Data, error) {
		return sampleData(), nil
	})

	req := httptest.NewRequest(http.MethodGet, "/data?start=2024-01-01T00:00:00Z", nil)
	req.Header.Set("Accept", "text/csv")
	rec := httptest.NewRecorder()

	Handle(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "text/csv", rec.Header().Get("Content-Type"))
}

func TestHandle_FetchError_Returns500(t *testing.T) {
	withFetchPoints(t, func(string) ([]*Data, error) {
		return nil, errFake
	})

	req := httptest.NewRequest(http.MethodGet, "/data?start=2024-01-01T00:00:00Z", nil)
	rec := httptest.NewRecorder()

	Handle(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestWriteCSV_HeaderRowAndColumnOrder(t *testing.T) {
	rec := httptest.NewRecorder()
	writeCSV(rec, nil)

	rows, err := csv.NewReader(bytes.NewReader(rec.Body.Bytes())).ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 1, "expected only the header row when no data is provided")
	require.Equal(t, csvHeaders, rows[0])
}

// fakeIterator drives collectPoints from a slice of pre-built FluxRecords.
type fakeIterator struct {
	records []*influxquery.FluxRecord
	idx     int
}

func (f *fakeIterator) Next() bool {
	if f.idx >= len(f.records) {
		return false
	}
	f.idx++
	return true
}

func (f *fakeIterator) Record() *influxquery.FluxRecord { return f.records[f.idx-1] }

func newRecord(field, host string, value interface{}, ts time.Time) *influxquery.FluxRecord {
	values := map[string]interface{}{
		"_field": field,
		"_time":  ts,
		"_value": value,
	}
	if host != "" {
		values["host"] = host
	}
	return influxquery.NewFluxRecord(0, values)
}

func TestCollectPoints_SkipsMissingHostTag(t *testing.T) {
	ts := time.Unix(100, 0).UTC()
	it := &fakeIterator{records: []*influxquery.FluxRecord{
		newRecord("co2", "", 400.0, ts),
	}}

	points := collectPoints(it)

	require.Empty(t, points, "records without a host tag should be skipped")
}

func TestCollectPoints_SkipsNilValue(t *testing.T) {
	ts := time.Unix(200, 0).UTC()
	it := &fakeIterator{records: []*influxquery.FluxRecord{
		newRecord("co2", "frog-01", nil, ts),
	}}

	points := collectPoints(it)

	require.Len(t, points, 1, "a placeholder Data entry is created before the nil-value check")
	for _, d := range points {
		require.Equal(t, "frog-01", d.Host)
		require.Equal(t, 0.0, d.CO2, "nil Value() must not be written into the field")
	}
}

func TestCollectPoints_PopulatesKnownFields(t *testing.T) {
	ts := time.Unix(300, 0).UTC()
	it := &fakeIterator{records: []*influxquery.FluxRecord{
		newRecord("co2", "frog-02", 412.0, ts),
		newRecord("humidity", "frog-02", 55.5, ts),
		newRecord("unknown_field", "frog-02", 1.0, ts),
	}}

	points := collectPoints(it)

	require.Len(t, points, 1)
	for _, d := range points {
		require.Equal(t, 412.0, d.CO2)
		require.Equal(t, 55.5, d.Humidity)
	}
}

// errFake is a sentinel used by TestHandle_FetchError_Returns500.
var errFake = &fakeErr{msg: "boom"}

type fakeErr struct{ msg string }

func (e *fakeErr) Error() string { return e.msg }
