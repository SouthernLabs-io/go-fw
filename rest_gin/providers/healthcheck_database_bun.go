package providers

import (
	"github.com/southernlabs-io/go-fw/rest_gin/middleware"
	"github.com/uptrace/bun"
)

type DatabaseBunHealthCheckProvider struct {
	db *bun.DB
}

var _ middleware.HealthCheckProvider = new(DatabaseBunHealthCheckProvider)

func NewDatabaseBunHealthCheckProvider(db *bun.DB) *DatabaseBunHealthCheckProvider {
	if db == nil {
		return nil
	}
	return &DatabaseBunHealthCheckProvider{
		db,
	}
}
func (p DatabaseBunHealthCheckProvider) GetName() string {
	return "DB"
}

func (p DatabaseBunHealthCheckProvider) HealthCheck() error {
	return p.db.Ping()
}
