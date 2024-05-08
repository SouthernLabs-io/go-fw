package redis

import (
	"context"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

type Redis struct {
	Client *redis.Client
}

func NewRedis(conf config.Config, lf *log.LoggerFactory) *Redis {
	if conf.Env.Type == config.EnvTypeTest {
		panic(errors.Newf(errors.ErrCodeBadState, "in a test: %+v", conf.Env))
	}

	rds := MustOpenRedis(conf, lf)

	return &Redis{
		Client: rds,
	}
}

func MustOpenRedis(conf config.Config, lf *log.LoggerFactory) *redis.Client {
	rdsConf := conf.Redis

	opt, err := redis.ParseURL(rdsConf.URL)

	if err != nil {
		panic(errors.Newf(errors.ErrCodeBadArgument, "failed to parse redis url: %s, error: %w", rdsConf.URL, err))
	}

	rds := redis.NewClient(opt)
	if err = rds.Ping(context.Background()).Err(); err != nil {
		panic(errors.NewUnknownf("failed to connect to redis: %w", err))
	}
	lf.GetLogger().Infof("Connected to redis: %s", rdsConf.URL)
	return rds
}

func (r Redis) HealthCheck() error {
	return r.Client.Ping(context.Background()).Err()
}

func OnStop(r Redis) error {
	return r.Client.Close()
}

var Module = fx.Provide(fx.Annotate(NewRedis, fx.OnStop(OnStop)))
