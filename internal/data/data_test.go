package data

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTransformURLToQuery(t *testing.T) {
	v := url.Values{
		"start":  []string{exampleTimeFmt},
		"stop":   []string{exampleTimeFmt},
		"hosts":  []string{"host"},
		"fields": []string{"field"},
	}

	query, err := transformURLToQuery(v)
	require.NoError(t, err)

	expected := fmt.Sprintf(`from(bucket:"co2") |> range(start:%s,stop:%s) |> filter(fn: (r) => r.host == "host") |> filter(fn: (r) => r._field == "field")`, exampleTimeFmt, exampleTimeFmt)
	require.Equal(t, expected, query)
}

func TestTransformURLToQuery_ErrNoStart(t *testing.T) {
	_, err := transformURLToQuery(url.Values{})
	require.Error(t, err)
}

func TestGetValues(t *testing.T) {
	a := time.Unix(0, 0)
	b := time.Unix(1, 0)

	m := map[string]*Data{"a": {Time: a}, "b": {Time: b}}
	v := getValues(m)

	require.Contains(t, v, &Data{Time: a})
	require.Contains(t, v, &Data{Time: b})
}
