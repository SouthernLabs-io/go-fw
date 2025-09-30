package healthcheck

import (
	database "github.com/southernlabs-io/go-fw/database/gorm"
	"go.uber.org/fx"
)

type DatabaseGORMHealthCheckProvider struct {
	db database.DB
}

var _ Provider = new(DatabaseGORMHealthCheckProvider)

func NewDatabaseGORMHealthCheckProvider(db database.DB) *DatabaseGORMHealthCheckProvider {
	if db.DB == nil {
		return nil
	}
	return &DatabaseGORMHealthCheckProvider{
		db,
	}
}

func NewDatabaseGORMHealthCheckProviderFx(params struct {
	fx.In

	DB database.DB `optional:"true"`
}) *DatabaseGORMHealthCheckProvider {
	return NewDatabaseGORMHealthCheckProvider(params.DB)
}

func (p DatabaseGORMHealthCheckProvider) GetName() string {
	return "DB"
}

func (p DatabaseGORMHealthCheckProvider) HealthCheck() error {
	return p.db.HealthCheck()
}
