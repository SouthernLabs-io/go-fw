package test

import (
	"context"

	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
	"github.com/southernlabs-io/go-fw/redis"
)

func NewTestRedis(conf config.Config, lf *log.LoggerFactory) redis.Redis {
	if conf.Env.Type != config.EnvTypeTest {
		panic(errors.Newf(errors.ErrCodeBadState, "not in a test: %+v", conf.Env))
	}

	client := redis.MustOpenRedis(conf, lf)
	return redis.Redis{
		Client: client,
	}
}

func OnTestRedisStop(redis redis.Redis) error {
	err := redis.Client.FlushDB(context.Background()).Err()
	if err != nil {
		panic(errors.NewUnknownf("failed to flush redis: %w", err))
	}
	return nil
}

var ModuleRedis = fx.Provide(
	fx.Annotate(
		NewTestRedis,
		fx.OnStop(OnTestRedisStop),
	),
)
