package providers

import (
	lib "github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/middlewares"
)

type DatabaseHealthCheckProvider struct {
	db lib.Database
}

var _ middlewares.HealthCheckProvider = new(DatabaseHealthCheckProvider)

func NewDatabaseHealthCheckProvider(db lib.Database) *DatabaseHealthCheckProvider {
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
