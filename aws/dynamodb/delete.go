package dynamodb

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

type Delete struct {
	Result ExecutionResult

	db    *DynamoDB
	model Model
}

func (d *DynamoDB) Delete(model Model) *Delete {
	return &Delete{
		db:    d,
		model: model,
	}
}

func (d *Delete) Execute(ctx context.Context) (ExecutionResult, error) {
	if d.Result.Err != nil {
		return ExecutionResult{}, d.Result.Err
	}
	tableName := d.db.tablePrefix + d.model.GetTableName()
	log.GetLoggerFromCtx(ctx).Debugf("Deleting item from table: %s", tableName)

	whereClause := ""
	var args []any
	pk := d.model.GetPK()
	if pk.RangeKey != "" {
		whereClause = fmt.Sprintf("%s = ? AND %s = ?", pk.HashKey, pk.RangeKey)
		args = append(args, pk.Hash, pk.Range)
	} else {
		whereClause = fmt.Sprintf("%s = ?", pk.HashKey)
		args = append(args, pk.Hash)
	}
	parameters, err := attributevalue.MarshalListWithOptions(args, d.db.encoderConfigurator)
	if err != nil {
		d.Result.Err = errors.NewUnknownf("failed to marshal parameters, error: %w", err)
		return ExecutionResult{}, d.Result.Err
	}

	res, err := d.db.client.ExecuteStatement(ctx, &dynamodb.ExecuteStatementInput{
		Statement:  aws.String(fmt.Sprintf("DELETE FROM %q WHERE %s RETURNING ALL OLD *", tableName, whereClause)),
		Parameters: parameters,
	})
	if res != nil {
		d.Result.Metadata = res.ResultMetadata
		d.Result.ConsumedCapacity = res.ConsumedCapacity
	}
	if err != nil || res == nil {
		var ce *types.ConditionalCheckFailedException
		if errors.As(err, &ce) {
			d.Result.Err = errors.Newf(errors.ErrCodeNotFound, "item not found in table: %s", tableName)
		} else {
			d.Result.Err = errors.NewUnknownf("failed to delete item from table: %s, error: %w", tableName, err)
		}
		return ExecutionResult{}, d.Result.Err
	}

	if len(res.Items) == 0 {
		d.Result.Err = errors.Newf(errors.ErrCodeNotFound, "item not found in table: %s", tableName)
	}

	return d.Result, d.Result.Err
}
