package healthcheck

import (
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type DatabaseBunHealthCheckProvider struct {
	db *bun.DB
}

var _ Provider = new(DatabaseBunHealthCheckProvider)

func NewDatabaseBunHealthCheckProvider(db *bun.DB) *DatabaseBunHealthCheckProvider {
	if db == nil {
		return nil
	}
	return &DatabaseBunHealthCheckProvider{
		db,
	}
}

func NewDatabaseBunHealthCheckProviderFx(params struct {
	fx.In

	DB *bun.DB `optional:"true"`
}) *DatabaseBunHealthCheckProvider {
	return NewDatabaseBunHealthCheckProvider(params.DB)
}

func (p DatabaseBunHealthCheckProvider) GetName() string {
	return "DB"
}

func (p DatabaseBunHealthCheckProvider) HealthCheck() error {
	return p.db.Ping()
}
