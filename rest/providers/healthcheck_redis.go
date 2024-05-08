package providers

import (
	"github.com/southernlabs-io/go-fw/redis"
	"github.com/southernlabs-io/go-fw/rest/middleware"
)

type RedisHealthCheckProvider struct {
	redis redis.Redis
}

var _ middleware.HealthCheckProvider = new(RedisHealthCheckProvider)

func NewRedisHealthCheckProvider(redis redis.Redis) *RedisHealthCheckProvider {
	if redis.Client == nil {
		return nil
	}
	return &RedisHealthCheckProvider{
		redis,
	}
}

func (p RedisHealthCheckProvider) GetName() string {
	return "Redis"
}

func (p RedisHealthCheckProvider) HealthCheck() error {
	return p.redis.HealthCheck()
}
