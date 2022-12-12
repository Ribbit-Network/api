package data

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type query struct {
	bucket   string
	start    time.Time
	stop     time.Time
	hosts    []string
	fields   []string
	interval time.Duration
}

func NewQuery(values url.Values) (string, error) {
	q := query{bucket: os.Getenv("INFLUXDB_BUCKET")}

	if val := values.Get("start"); val != "" {
		start, err := time.Parse(time.RFC3339, val)
		if err != nil {
			return "", err
		}
		q.start = start
	} else {
		return "", fmt.Errorf(`missing required parameter: "start"`)
	}

	if val := values.Get("stop"); val != "" {
		stop, err := time.Parse(time.RFC3339, val)
		if err != nil {
			return "", err
		}
		q.stop = stop
	}

	if val := values.Get("hosts"); val != "" {
		q.hosts = strings.Split(val, ",")
	}

	if val := values.Get("fields"); val != "" {
		q.fields = strings.Split(val, ",")
	}

	if val := values.Get("interval"); val != "" {
		interval, err := time.ParseDuration(val)
		if err != nil {
			return "", err
		}
		q.interval = interval
	}

	return q.String(), nil
}

func (q query) String() string {
	x := []string{
		fmt.Sprintf(`from(bucket:"%s")`, q.bucket),
		buildRange(q.start, q.stop),
	}

	if len(q.hosts) > 0 {
		x = append(x, buildConditionFilter("host", q.hosts))
	}

	if len(q.fields) > 0 {
		x = append(x, buildConditionFilter("_field", q.fields))
	}

	if q.interval > 0 {
		x = append(x, fmt.Sprintf("aggregateWindow(every: %s, fn: mean, createEmpty: false)", q.interval))
	}

	return strings.Join(x, " |> ")
}

func buildRange(start, stop time.Time) string {
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
