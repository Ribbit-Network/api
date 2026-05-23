package sensors

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	influxquery "github.com/influxdata/influxdb-client-go/v2/api/query"
	"github.com/stretchr/testify/require"
)

func withFetchSensors(t *testing.T, fn func() ([]string, error)) {
	t.Helper()
	orig := fetchSensors
	fetchSensors = fn
	t.Cleanup(func() { fetchSensors = orig })
}

func TestHandle_MethodNotAllowed(t *testing.T) {
	withFetchSensors(t, func() ([]string, error) {
		t.Fatal("fetchSensors should not be called for non-GET")
		return nil, nil
	})

	req := httptest.NewRequest(http.MethodPost, "/sensors", nil)
	rec := httptest.NewRecorder()

	Handle(rec, req)

	require.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

func TestHandle_ReturnsJSON(t *testing.T) {
	withFetchSensors(t, func() ([]string, error) {
		return []string{"frog-01", "frog-02"}, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/sensors", nil)
	rec := httptest.NewRecorder()

	Handle(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	var payload struct {
		Sensors []string `json:"sensors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
	require.Equal(t, []string{"frog-01", "frog-02"}, payload.Sensors)
}

func TestHandle_EmptyResult_ReturnsEmptyArray(t *testing.T) {
	withFetchSensors(t, func() ([]string, error) {
		return nil, nil
	})

	req := httptest.NewRequest(http.MethodGet, "/sensors", nil)
	rec := httptest.NewRecorder()

	Handle(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"sensors":[]`, "nil result must serialize as [], not null")
}

func TestHandle_FetchError_Returns500_QueryFailedBody(t *testing.T) {
	withFetchSensors(t, func() ([]string, error) {
		return nil, fmt.Errorf("%w: %v", errQueryFailed, errFake)
	})

	req := httptest.NewRequest(http.MethodGet, "/sensors", nil)
	rec := httptest.NewRecorder()

	Handle(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, "query failed", strings.TrimSpace(rec.Body.String()))
}

func TestHandle_IteratorError_Returns500_QueryResultErrorBody(t *testing.T) {
	withFetchSensors(t, func() ([]string, error) {
		return nil, fmt.Errorf("%w: %v", errQueryResult, errFake)
	})

	req := httptest.NewRequest(http.MethodGet, "/sensors", nil)
	rec := httptest.NewRecorder()

	Handle(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Equal(t, "query result error", strings.TrimSpace(rec.Body.String()))
}

func TestBuildQuery(t *testing.T) {
	got := buildQuery("frog_fleet")
	require.Contains(t, got, `import "influxdata/influxdb/schema"`)
	require.Contains(t, got, `schema.tagValues(bucket: "frog_fleet", tag: "host")`)
}

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

func TestCollectIDs_PullsStringValues(t *testing.T) {
	it := &fakeIterator{records: []*influxquery.FluxRecord{
		influxquery.NewFluxRecord(0, map[string]interface{}{"_value": "frog-01"}),
		influxquery.NewFluxRecord(0, map[string]interface{}{"_value": "frog-02"}),
	}}

	require.Equal(t, []string{"frog-01", "frog-02"}, collectIDs(it))
}

func TestCollectIDs_SkipsNonStringValues(t *testing.T) {
	it := &fakeIterator{records: []*influxquery.FluxRecord{
		influxquery.NewFluxRecord(0, map[string]interface{}{"_value": "frog-01"}),
		influxquery.NewFluxRecord(0, map[string]interface{}{"_value": 42}),
		influxquery.NewFluxRecord(0, map[string]interface{}{"_value": nil}),
	}}

	require.Equal(t, []string{"frog-01"}, collectIDs(it))
}

var errFake = errors.New("boom")
