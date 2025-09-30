package providers

import (
	database "github.com/southernlabs-io/go-fw/database/gorm"
	"github.com/southernlabs-io/go-fw/rest_gin/middleware"
)

type DatabaseGORMHealthCheckProvider struct {
	db database.DB
}

var _ middleware.HealthCheckProvider = new(DatabaseGORMHealthCheckProvider)

func NewDatabaseGORMHealthCheckProvider(db database.DB) *DatabaseGORMHealthCheckProvider {
	if db.DB == nil {
		return nil
	}
	return &DatabaseGORMHealthCheckProvider{
		db,
	}
}
func (p DatabaseGORMHealthCheckProvider) GetName() string {
	return "DB"
}

func (p DatabaseGORMHealthCheckProvider) HealthCheck() error {
	return p.db.HealthCheck()
}
