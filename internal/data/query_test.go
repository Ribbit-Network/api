package data

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	exampleTimeFmt = "1970-01-01T00:00:01Z"
	exampleTime, _ = time.Parse(time.RFC3339, exampleTimeFmt)
)

func TestString_RangeFilter(t *testing.T) {
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

func TestBuildRangeFilter_StartAndStop(t *testing.T) {
	expected := fmt.Sprintf("range(start:%s,stop:%s)", exampleTimeFmt, exampleTimeFmt)
	require.Equal(t, expected, buildRangeFilter(exampleTime, exampleTime))
}

func TestBuildRangeFilter_Start(t *testing.T) {
	expected := fmt.Sprintf("range(start:%s)", exampleTimeFmt)
	require.Equal(t, expected, buildRangeFilter(exampleTime, time.Time{}))
}

func TestBuildConditionFilter(t *testing.T) {
	expected := `filter(fn: (r) => r.key == "a" or r.key == "b")`
	require.Equal(t, expected, buildConditionFilter("key", []string{"a", "b"}))
}
