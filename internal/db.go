package internal

import (
	"context"
	"os"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type db struct {
	influxdb2.Client
}

func NewDB() *db {
	serverUrl := os.Getenv("INFLUXDB_SERVER_URL")
	authToken := os.Getenv("INFLUXDB_AUTH_TOKEN")

	return &db{influxdb2.NewClient(serverUrl, authToken)}
}

func (d *db) Query(q string) (*api.QueryTableResult, error) {
	org := os.Getenv("INFLUXDB_ORG")
	ctx := context.Background()

	return d.Client.QueryAPI(org).Query(ctx, q)
}
