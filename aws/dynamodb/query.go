package dynamodb

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/southernlabs-io/go-fw/functional/slices"
	"github.com/southernlabs-io/go-fw/log"

	"github.com/southernlabs-io/go-fw/errors"
)

// Query is a builder for querying items in a DynamoDB table using PartiQL.
type Query struct {
	ExecuteStatementInput dynamodb.ExecuteStatementInput
	WhereClause           string
	WhereArgs             []any
	AllowScan             bool

	Result ExecutionResult

	single bool
	db     *DynamoDB
	model  Model
	target any
}

// WithConsistentRead sets the query to use consistent read. By default, queries use eventually consistent read.
func (q *Query) WithConsistentRead() *Query {
	q.ExecuteStatementInput.ConsistentRead = aws.Bool(true)
	return q

}

// Where sets the where clause and arguments for the query. It must be written in PartiQL syntax with the field names
// being the same as the DynamoDB table. There is no named parameters support.
// Caution: Calling this function will override any previous where clause and arguments
// Example:
//
//	Where("id = ? AND name = ?", "123", "John")
//
// Refer to: https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/ql-reference.html
func (q *Query) Where(where string, args ...any) *Query {
	q.WhereClause = where
	q.WhereArgs = args

	return q
}

// WithAllowScan allows the query to perform a scan operation. By default, scan operations are not allowed.
func (q *Query) WithAllowScan() *Query {
	q.AllowScan = true
	return q
}

var fieldEqRegex = regexp.MustCompile(`(^|\s+)(\w+)\s*=\s*\S+`)
var fieldInRegex = regexp.MustCompile(`(?i)(^|\s+)(\w+)\s+IN\s+\[\s*\S+.*?]`)

func matchFieldRegex(regex *regexp.Regexp, field string, clause string) bool {
	idx := regex.FindAllStringSubmatchIndex(clause, -1)
	if len(idx) == 0 {
		return false
	}

	for _, match := range idx {
		if clause[match[4]:match[5]] == field {
			return true
		}
	}

	return false
}

func (q *Query) checkForScan() {
	if !q.AllowScan {
		pk := q.model.GetPK()
		if !matchFieldRegex(fieldEqRegex, pk.HashKey, q.WhereClause) && !matchFieldRegex(fieldInRegex, pk.HashKey, q.WhereClause) {
			q.Result.Err = errors.Newf(
				errors.ErrCodeBadArgument,
				"scan operation is not allowed, where clause must contain a restriction on the pk.hash field",
			)
		}
	}
}

// Get returns a new Query builder for the provided target model. It will expect a single result.
func (d *DynamoDB) Get(target Model) *Query {
	q := &Query{
		db:     d,
		single: true,
		model:  target,
		target: target,
	}

	pk := target.GetPK()
	if pk.RangeKey != "" {
		q.Where(fmt.Sprintf("%s = ? AND %s = ?", pk.HashKey, pk.RangeKey), pk.Hash, pk.Range)
	} else {
		q.Where(fmt.Sprintf("%s = ?", pk.HashKey), pk.Hash)
	}

	return q
}

// Query returns a new Query builder for the provided target slice.
// Passing a prefilled slice with elements will result in a query by PK.
func (d *DynamoDB) Query(targetSlice any) *Query {
	q := &Query{
		db:     d,
		target: targetSlice,
	}

	sliceType := reflect.TypeOf(targetSlice).Elem()
	if sliceType.Kind() != reflect.Slice {
		q.Result.Err = errors.Newf(errors.ErrCodeBadArgument, "expected slice, got %s", sliceType.Kind())
		return q
	}
	sliceValue := reflect.ValueOf(targetSlice).Elem()
	if sliceValue.Len() == 0 {
		q.model = reflect.New(sliceType.Elem()).Interface().(Model)
	} else {
		pks := make([]PrimaryKey, sliceValue.Len())
		q.model = sliceValue.Index(0).Addr().Interface().(Model)
		for i := 0; i < sliceValue.Len(); i++ {
			elem := sliceValue.Index(i).Addr().Interface()
			pks[i] = elem.(Model).GetPK()
		}
		q.WithPKs(pks...)
	}

	return q
}

func (q *Query) WithPKs(pks ...PrimaryKey) *Query {
	if len(pks) == 0 {
		return q
	}

	pk := q.model.GetPK()
	if pk.RangeKey != "" {
		q.Where(
			fmt.Sprintf("%s IN [%s] AND %s IN [%s]",
				pk.HashKey,
				strings.TrimSuffix(strings.Repeat("?,", len(pks)), ","),
				pk.RangeKey,
				strings.TrimSuffix(strings.Repeat("?,", len(pks)), ","),
			),
			append(slices.Map(pks, func(item PrimaryKey) any {
				return item.Hash
			}), slices.Map(pks, func(item PrimaryKey) any {
				return item.Range
			})...))
	} else {
		q.Where(
			fmt.Sprintf("%s IN [%s]",
				pk.HashKey,
				strings.TrimSuffix(strings.Repeat("?,", len(pks)), ","),
			), slices.Map(pks, func(item PrimaryKey) any {
				return item.Hash
			})...)
	}
	return q
}

func (q *Query) Execute(ctx context.Context) (ExecutionResult, error) {
	q.checkForScan()
	if q.Result.Err != nil {
		return ExecutionResult{}, q.Result.Err
	}

	tableName := q.db.tablePrefix + q.model.GetTableName()
	log.GetLoggerFromCtx(ctx).Debugf("Querying table: %s", tableName)

	// Marshal parameters
	parameters, err := attributevalue.MarshalListWithOptions(q.WhereArgs, q.db.encoderConfigurator)
	if err != nil {
		q.Result.Err = errors.NewUnknownf("failed to marshal parameters, error: %w", err)
		return ExecutionResult{}, q.Result.Err
	}
	statement := fmt.Sprintf("SELECT * FROM %q", tableName)
	if q.WhereClause != "" {
		statement += " WHERE " + q.WhereClause
	}
	q.ExecuteStatementInput.Statement = aws.String(statement)
	q.ExecuteStatementInput.Parameters = parameters

	// Execute
	res, err := q.db.client.ExecuteStatement(ctx, &q.ExecuteStatementInput)
	if res != nil {
		q.Result.Metadata = res.ResultMetadata
		q.Result.ConsumedCapacity = res.ConsumedCapacity
	}
	// res == nil is to avoid the compiler complaining about res being potentially nil after this block
	if err != nil || res == nil {
		return ExecutionResult{}, errors.NewUnknownf("failed to query table: %s, error: %w", tableName, err)
	}

	if q.single {
		if len(res.Items) == 0 {
			q.Result.Err = errors.Newf(errors.ErrCodeNotFound, "item not found in table: %s", tableName)
		} else if len(res.Items) > 1 {
			q.Result.Err = errors.Newf(errors.ErrCodeBadState, "expected 1 item, got %d", len(res.Items))
		} else {
			if err = attributevalue.UnmarshalMapWithOptions(res.Items[0], q.target, q.db.decoderConfigurator); err != nil {
				q.Result.Err = errors.NewUnknownf("failed to unmarshal item, error: %w", err)
			}
		}
	} else {
		if err = attributevalue.UnmarshalListOfMapsWithOptions(res.Items, q.target, q.db.decoderConfigurator); err != nil {
			q.Result.Err = errors.NewUnknownf("failed to unmarshal items, error: %w", err)
		}
	}

	return q.Result, q.Result.Err
}
