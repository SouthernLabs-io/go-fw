package healthcheck

import (
	"github.com/southernlabs-io/go-fw/redis"
	"go.uber.org/fx"
)

type RedisHealthCheckProvider struct {
	redis redis.Redis
}

var _ Provider = new(RedisHealthCheckProvider)

func NewRedisHealthCheckProvider(redis redis.Redis) *RedisHealthCheckProvider {
	if redis.Client == nil {
		return nil
	}
	return &RedisHealthCheckProvider{
		redis,
	}
}

func NewRedisHealthCheckProviderFx(params struct {
	fx.In

	Redis redis.Redis `optional:"true"`
}) *RedisHealthCheckProvider {
	return NewRedisHealthCheckProvider(params.Redis)
}

func (p RedisHealthCheckProvider) GetName() string {
	return "Redis"
}

func (p RedisHealthCheckProvider) HealthCheck() error {
	return p.redis.HealthCheck()
}
