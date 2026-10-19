package database_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/database"
)

type fakePool struct {
	calls []string
}

func (f *fakePool) SetMaxOpenConns(int)              { f.calls = append(f.calls, "open") }
func (f *fakePool) SetMaxIdleConns(int)              { f.calls = append(f.calls, "idle") }
func (f *fakePool) SetConnMaxIdleTime(time.Duration) { f.calls = append(f.calls, "idleTime") }
func (f *fakePool) SetConnMaxLifetime(time.Duration) { f.calls = append(f.calls, "lifetime") }

func TestApplyPoolConfig_NotSet_KeepsDefaults(t *testing.T) {
	pool := &fakePool{}
	database.ApplyPoolConfig(pool, config.DatabaseConfig{})
	require.Empty(t, pool.calls)
}

func TestApplyPoolConfig_OnlyAppliesSetValues(t *testing.T) {
	pool := &fakePool{}
	database.ApplyPoolConfig(pool, config.DatabaseConfig{ConnMaxLifetime: time.Minute})
	require.Equal(t, []string{"lifetime"}, pool.calls)
}

func TestApplyPoolConfig_AllSet(t *testing.T) {
	pool := &fakePool{}
	database.ApplyPoolConfig(pool, config.DatabaseConfig{
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxIdleTime: time.Minute,
		ConnMaxLifetime: time.Hour,
	})
	require.Equal(t, []string{"open", "idle", "idleTime", "lifetime"}, pool.calls)
}
