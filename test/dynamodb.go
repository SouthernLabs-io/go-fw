package test

import (
	"context"
	"testing"

	awsdynamodb "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"go.uber.org/fx"

	"github.com/southernlabs-io/go-fw/aws/dynamodb"
	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

func NewTestDynamoDB(_ testing.TB, ctx context.Context, conf config.Config, dynamoDBClient *awsdynamodb.Client) *dynamodb.DynamoDB {
	if conf.Env.Type != config.EnvTypeTest {
		panic(errors.Newf(errors.ErrCodeBadState, "not in a test: %+v", conf.Env))
	}

	ddb := dynamodb.NewDynamoDB(conf, dynamoDBClient)

	// Delete all existing tables
	err := ddb.DeleteAllTables(ctx)
	if err != nil {
		panic(err)
	}
	return ddb
}

func OnTestDynamoDBStop(tb testing.TB, ctx context.Context, ddb *dynamodb.DynamoDB) error {
	if tb.Failed() {
		log.GetLoggerFromCtx(ctx).Warn("Test failed, not deleting tables")
		return nil
	}
	return ddb.DeleteAllTables(ctx)
}

var ModuleTestDynamoDB = fx.Options(
	fx.Provide(dynamodb.NewAWSDynamoDBClient),
	fx.Provide(fx.Annotate(
		NewTestDynamoDB,
		fx.OnStop(OnTestDynamoDBStop),
	)),
)
