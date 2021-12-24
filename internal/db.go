package internal

import (
	"context"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

const (
	serverURL = "https://us-west-2-1.aws.cloud2.influxdata.com"
	authToken = "***REMOVED***"
	org       = "keenan.johnson@gmail.com"
)

type db struct {
	influxdb2.Client
}

func NewDB() *db {
	return &db{influxdb2.NewClient(serverURL, authToken)}
}

func (d *db) Query(q string) (*api.QueryTableResult, error) {
	queryAPI := d.Client.QueryAPI(org)
	ctx := context.Background()
	return queryAPI.Query(ctx, q)
}
