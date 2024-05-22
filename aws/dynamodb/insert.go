package dynamodb

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"

	"github.com/southernlabs-io/go-fw/aws/partiql"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

type InsertOne struct {
	Result ExecutionResult

	db    *DynamoDB
	model Model
}

func (d *DynamoDB) Insert(model Model) *InsertOne {
	// First fill time values as a model can use them as range key
	var zero time.Time
	if model.GetCreateTs() == zero {
		now := d.NowFunc()
		model.SetCreateTs(now)
	}
	if updatable, is := model.(ModelWithUpdateTs); is {
		updatable.SetUpdateTs(model.GetCreateTs())
	}

	pk := model.GetPK()
	if pk.Hash == "" {
		// We can only auto-generate the hash key
		if pk.RangeKey != "" && pk.Range == nil {
			return &InsertOne{
				Result: ExecutionResult{
					Err: errors.Newf(errors.ErrCodeBadArgument, "missing range key for model"),
				},
			}
		}
		pk.Hash = uuid.New().String()
		model.SetPK(pk)
	}

	return &InsertOne{
		db:    d,
		model: model,
	}
}

func (i *InsertOne) Execute(ctx context.Context) (ExecutionResult, error) {
	if i.Result.Err != nil {
		return ExecutionResult{}, i.Result.Err
	}

	tableName := i.db.tablePrefix + i.model.GetTableName()
	log.GetLoggerFromCtx(ctx).Debugf("Inserting item in table: %s", tableName)

	item, err := partiql.NewEncoder().WithTimeMarshaler(i.db.TimeMarshalerFunc).Marshal(i.model)
	if err != nil {
		i.Result.Err = errors.NewUnknownf("failed to marshal item, error: %w", err)
		return ExecutionResult{}, i.Result.Err
	}

	res, err := i.db.client.ExecuteStatement(ctx, &dynamodb.ExecuteStatementInput{
		Statement: aws.String(fmt.Sprintf("INSERT INTO %q VALUE %s", tableName, item)),
	})
	if err != nil {
		var die *types.DuplicateItemException
		if errors.As(err, &die) {
			i.Result.Err = errors.Newf(errors.ErrCodeConflict, "item already exists in table: %s", tableName)
		} else {
			//TODO: docs says it should be a types.DuplicateItemException, but it's not working. This is what we get now
			var gae *smithy.GenericAPIError
			if errors.As(err, &gae) && gae.Code == "DuplicateItem" {
				i.Result.Err = errors.Newf(errors.ErrCodeConflict, "item already exists in table: %s", tableName)
			} else {
				i.Result.Err = errors.NewUnknownf("failed to put item in table: %s, error: %w", tableName, err)
			}
		}
		return ExecutionResult{}, i.Result.Err
	}

	i.Result.Metadata = res.ResultMetadata
	i.Result.ConsumedCapacity = res.ConsumedCapacity

	return i.Result, i.Result.Err
}
