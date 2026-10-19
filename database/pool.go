package database

import (
	"time"

	"github.com/southernlabs-io/go-fw/config"
)

// PoolSetter is implemented by *sql.DB (and therefore by *gorm's sql.DB) and *bun.DB.
type PoolSetter interface {
	SetMaxOpenConns(n int)
	SetMaxIdleConns(n int)
	SetConnMaxIdleTime(d time.Duration)
	SetConnMaxLifetime(d time.Duration)
}

// ApplyPoolConfig applies the pool settings from the config. Zero values are skipped, so anything not
// configured keeps the database/sql default (e.g. 2 idle connections instead of none).
func ApplyPoolConfig(db PoolSetter, conf config.DatabaseConfig) {
	if conf.MaxOpenConns > 0 {
		db.SetMaxOpenConns(conf.MaxOpenConns)
	}
	if conf.MaxIdleConns > 0 {
		db.SetMaxIdleConns(conf.MaxIdleConns)
	}
	if conf.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(conf.ConnMaxIdleTime)
	}
	if conf.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(conf.ConnMaxLifetime)
	}
}
