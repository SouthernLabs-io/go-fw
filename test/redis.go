package test

import (
	"context"

	"go.uber.org/fx"

	lib "github.com/southernlabs-io/go-fw/core"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/redis"
)

func NewTestRedis(conf lib.Config, lf *lib.LoggerFactory) redis.Redis {
	if conf.Env.Type != lib.EnvTypeTest {
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

var TestModuleRedis = fx.Provide(
	fx.Annotate(
		NewTestRedis,
		fx.OnStop(OnTestRedisStop),
	),
)
