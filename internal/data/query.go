package data

import (
	"fmt"
	"strings"
	"time"
)

type query struct {
	start  time.Time
	stop   time.Time
	hosts  []string
	fields []string
}

func (q query) String() string {
	x := []string{
		`from(bucket:"co2")`,
		buildRangeFilter(q.start, q.stop),
	}

	if len(q.hosts) > 0 {
		x = append(x, buildConditionFilter("host", q.hosts))
	}

	if len(q.fields) > 0 {
		x = append(x, buildConditionFilter("_field", q.fields))
	}

	return strings.Join(x, " |> ")
}

func buildRangeFilter(start, stop time.Time) string {
	if stop.Unix() > 0 {
		return fmt.Sprintf("range(start:%s,stop:%s)", start.Format(time.RFC3339), stop.Format(time.RFC3339))
	} else {
		return fmt.Sprintf("range(start:%s)", start.Format(time.RFC3339))
	}
}

func buildConditionFilter(key string, values []string) string {
	conditions := make([]string, len(values))
	for i, val := range values {
		conditions[i] = fmt.Sprintf(`r.%s == "%s"`, key, val)
	}
	return fmt.Sprintf("filter(fn: (r) => %s)", strings.Join(conditions, " or "))
}
