package data

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	exampleTimeFmt = "1970-01-01T00:00:01Z"
	exampleTime, _ = time.Parse(time.RFC3339, exampleTimeFmt)
)

func TestNewQuery(t *testing.T) {
	v := url.Values{
		"start":  []string{exampleTimeFmt},
		"stop":   []string{exampleTimeFmt},
		"hosts":  []string{"host"},
		"fields": []string{"field"},
	}

	query, err := NewQuery(v)
	require.NoError(t, err)

	expected := fmt.Sprintf(`from(bucket:"co2") |> range(start:%s,stop:%s) |> filter(fn: (r) => r.host == "host") |> filter(fn: (r) => r._field == "field")`, exampleTimeFmt, exampleTimeFmt)
	require.Equal(t, expected, query)
}

func TestNewQuery_ErrNoStart(t *testing.T) {
	_, err := NewQuery(url.Values{})
	require.Error(t, err)
}

func TestString_Range(t *testing.T) {
	expected := fmt.Sprintf(`from(bucket:"co2") |> range(start:%s)`, exampleTimeFmt)
	require.Equal(t, expected, query{start: exampleTime}.String())
}

func TestString_HostFilter(t *testing.T) {
	expected := `from(bucket:"co2") |> range(start:0001-01-01T00:00:00Z) |> filter(fn: (r) => r.host == "host")`
	require.Equal(t, expected, query{hosts: []string{"host"}}.String())
}

func TestString_FieldFilter(t *testing.T) {
	expected := `from(bucket:"co2") |> range(start:0001-01-01T00:00:00Z) |> filter(fn: (r) => r._field == "field")`
	require.Equal(t, expected, query{fields: []string{"field"}}.String())
}

func TestString_AggregateWindow(t *testing.T) {
	expected := `from(bucket:"co2") |> range(start:0001-01-01T00:00:00Z) |> aggregateWindow(every: 1s, fn: mean, createEmpty: false)`
	require.Equal(t, expected, query{interval: time.Second}.String())
}

func TestBuildRange_StartAndStop(t *testing.T) {
	expected := fmt.Sprintf("range(start:%s,stop:%s)", exampleTimeFmt, exampleTimeFmt)
	require.Equal(t, expected, buildRange(exampleTime, exampleTime))
}

func TestBuildRange_Start(t *testing.T) {
	expected := fmt.Sprintf("range(start:%s)", exampleTimeFmt)
	require.Equal(t, expected, buildRange(exampleTime, time.Time{}))
}

func TestBuildConditionFilter(t *testing.T) {
	expected := `filter(fn: (r) => r.key == "a" or r.key == "b")`
	require.Equal(t, expected, buildConditionFilter("key", []string{"a", "b"}))
}
