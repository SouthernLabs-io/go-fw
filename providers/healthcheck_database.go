package providers

import (
	"github.com/southernlabs-io/go-fw/database"
	"github.com/southernlabs-io/go-fw/rest/middlewares"
)

type DatabaseHealthCheckProvider struct {
	db database.DB
}

var _ middlewares.HealthCheckProvider = new(DatabaseHealthCheckProvider)

func NewDatabaseHealthCheckProvider(db database.DB) *DatabaseHealthCheckProvider {
	if db.DB == nil {
		return nil
	}
	return &DatabaseHealthCheckProvider{
		db,
	}
}
func (p DatabaseHealthCheckProvider) GetName() string {
	return "DB"
}

func (p DatabaseHealthCheckProvider) HealthCheck() error {
	return p.db.HealthCheck()
}
