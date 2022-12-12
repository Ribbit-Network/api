package data

import (
	"testing"
	"time"

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
