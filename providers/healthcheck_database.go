package providers

import (
	"github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/middlewares"
)

type DatabaseHealthCheckProvider struct {
	db core.Database
}

var _ middlewares.HealthCheckProvider = new(DatabaseHealthCheckProvider)

func NewDatabaseHealthCheckProvider(db core.Database) *DatabaseHealthCheckProvider {
	if db.DB == nil {
		return nil
	}
	return &DatabaseHealthCheckProvider{
		db,
	}
}
func (p DatabaseHealthCheckProvider) GetName() string {
	return "Database"
}

func (p DatabaseHealthCheckProvider) HealthCheck() error {
	return p.db.HealthCheck()
}
