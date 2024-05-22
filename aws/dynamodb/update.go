package dynamodb

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"

	"github.com/southernlabs-io/go-fw/aws/partiql"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/log"
)

// Update is a builder for updating items in a DynamoDB table using PartiQL.
// AWS docs: https://docs.aws.amazon.com/amazondynamodb/latest/developerguide/ql-reference.update.html
type Update struct {
	Result ExecutionResult

	db     *DynamoDB
	target Model

	setFields    map[string]any
	removeFields []string
}

// Update returns a new Update builder for the provided target model.
// The target model must have its primary key set.
func (d *DynamoDB) Update(target Model) *Update {
	pk := target.GetPK()
	if pk == (PrimaryKey{}) {
		return &Update{
			Result: ExecutionResult{
				Err: errors.Newf(errors.ErrCodeBadArgument, "missing pk.hash for update"),
			},
		}
	}
	if pk.RangeKey != "" && pk.Range == nil {
		return &Update{
			Result: ExecutionResult{
				Err: errors.Newf(errors.ErrCodeBadArgument, "missing pk.range for update"),
			},
		}
	}
	return &Update{
		db:     d,
		target: target,
	}
}

type collection struct {
	funcName string
	bag      bool
	items    []any
}

func newListAppend(items ...any) collection {
	return collection{funcName: "LIST_APPEND", items: items}
}

func newSetAdd(items ...any) collection {
	return collection{funcName: "SET_ADD", items: items, bag: true}
}

func newSetDelete(items ...any) collection {
	return collection{funcName: "SET_DELETE", items: items, bag: true}
}

type number interface {
	int8 | int16 | int32 | int64 | int | float32 | float64
}
type increment struct {
	field      string
	intValue   *int64
	uintValue  *uint64
	floatValue *float64
}

func newIncrementInt(field string, value int64) increment {
	return increment{field: field, intValue: &value}
}

func newIncrementUint(field string, value uint64) increment {
	return increment{field: field, uintValue: &value}
}

func newIncrementFloat(field string, value float64) increment {
	return increment{field: field, floatValue: &value}
}

// Set sets a field in the item. Note: the provided field name must be the same as in the partiql tag.
func (u *Update) Set(field string, value any) *Update {
	if u.setFields == nil {
		u.setFields = make(map[string]any)
	}
	if _, ok := u.setFields[field]; ok {
		u.Result.Err = errors.Newf(errors.ErrCodeBadArgument, "field already set: %s", field)
		return u
	}
	u.setFields[field] = value
	return u
}

// Remove removes a field from the item. Note: the provided field name must be the same as in the partiql tag.
func (u *Update) Remove(field string) *Update {
	u.removeFields = append(u.removeFields, field)
	return u
}

// AppendToList appends values to a list field. Note: the provided field name must be the same as in the partiql tag.
func (u *Update) AppendToList(field string, values ...any) *Update {
	u.Set(field, newListAppend(values...))
	return u
}

// RemoveFromList removes an item from a list field. Note: the provided field name must be the same as in the partiql tag.
func (u *Update) RemoveFromList(field string, idx int) *Update {
	u.removeFields = append(u.removeFields, fmt.Sprintf("%s[%d]", field, idx))
	return u
}

// AddToSet adds values to a set field. Note: the provided field name must be the same as in the partiql tag.
func (u *Update) AddToSet(field string, values ...any) *Update {
	u.Set(field, newSetAdd(values...))
	return u
}

// RemoveFromSet removes values from a set field. Note: the provided field name must be the same as in the partiql tag.
func (u *Update) RemoveFromSet(field string, values ...any) *Update {
	u.Set(field, newSetDelete(values...))
	return u
}

// Increment increments or decrements a numeric field. Note: the provided field name must be the same as in the partiql tag.
func (u *Update) Increment(field string, value any) *Update {
	switch v := value.(type) {
	case int, int8, int16, int32, int64:
		u.Set(field, newIncrementInt(field, reflect.ValueOf(v).Int()))
	case uint, uint8, uint16, uint32, uint64:
		u.Set(field, newIncrementUint(field, reflect.ValueOf(v).Uint()))
	case float32, float64:
		u.Set(field, newIncrementFloat(field, reflect.ValueOf(v).Float()))
	default:
		u.Result.Err = errors.Newf(errors.ErrCodeBadArgument, "invalid increment value: %v", value)
	}
	return u
}

// Execute executes the update operation. It will return immediately if there is an error in the builder.
func (u *Update) Execute(ctx context.Context) (ExecutionResult, error) {
	if u.Result.Err != nil {
		return ExecutionResult{}, u.Result.Err
	}

	tableName := u.db.tablePrefix + u.target.GetTableName()
	log.GetLoggerFromCtx(ctx).Debugf("Updating item in table: %s", tableName)

	parameters := make([]any, 0, 1+len(u.setFields)+len(u.removeFields))
	sb := strings.Builder{}
	sb.WriteString("UPDATE \"")
	sb.WriteString(tableName)
	sb.WriteString("\"")

	if updatable, is := u.target.(ModelWithUpdateTs); is {
		sb.WriteString(" SET ")
		sb.WriteString(updatable.GetUpdateTsKey())
		sb.WriteString(" = ?")
		parameters = append(parameters, u.db.NowFunc())
	}
	for field, value := range u.setFields {
		sb.WriteString(" SET ")
		sb.WriteString(field)
		sb.WriteString(" = ")
		switch v := value.(type) {
		case collection:
			sb.WriteString(v.funcName)
			sb.WriteByte('(')
			if v.bag {
				sb.WriteString(field)
				sb.WriteString(", ")
			} else {
				// Lists require special treatment when the field does not exist
				sb.WriteString("if_not_exists(")
				sb.WriteString(field)
				sb.WriteString(", []), ")
			}
			enc := partiql.NewEncoder().WithTimeMarshaler(u.db.TimeMarshalerFunc)
			bytes, err := enc.MarshalCollection(v.items, v.bag)
			if err != nil {
				u.Result.Err = errors.NewUnknownf("failed to encode value, error: %w", err)
				return ExecutionResult{}, u.Result.Err
			}
			sb.Write(bytes)
			sb.WriteByte(')')
		case increment:
			applyIncrement(&sb, field, v, &parameters)
		default:
			sb.WriteString("?")
			parameters = append(parameters, value)
		}
	}
	for _, field := range u.removeFields {
		sb.WriteString(" REMOVE ")
		sb.WriteString(field)
	}
	sb.WriteString(" WHERE ")
	pk := u.target.GetPK()
	sb.WriteString(pk.HashKey)
	sb.WriteString(" = ?")
	parameters = append(parameters, pk.Hash)
	if pk.RangeKey != "" {
		sb.WriteString(" AND ")
		sb.WriteString(pk.RangeKey)
		sb.WriteString(" = ?")
		parameters = append(parameters, pk.Range)
	}
	sb.WriteString(" RETURNING ALL NEW *")

	input := &dynamodb.ExecuteStatementInput{
		Statement: aws.String(sb.String()),
	}
	// Marshal parameters
	var err error
	input.Parameters, err = attributevalue.MarshalListWithOptions(parameters, u.db.encoderConfigurator)
	if err != nil {
		u.Result.Err = errors.NewUnknownf("failed to marshal parameters, error: %w", err)
		return ExecutionResult{}, u.Result.Err
	}

	// Execute
	res, err := u.db.client.ExecuteStatement(ctx, input)
	if res != nil {
		u.Result.Metadata = res.ResultMetadata
		u.Result.ConsumedCapacity = res.ConsumedCapacity
	}
	if err != nil {
		u.Result.Err = errors.NewUnknownf("failed to update item in table: %s, error: %w", u.db.tablePrefix+u.target.GetTableName(), err)
		return ExecutionResult{}, u.Result.Err
	}

	if len(res.Items) == 0 {
		u.Result.Err = errors.Newf(errors.ErrCodeNotFound, "item not found in table: %s", tableName)
	} else if len(res.Items) > 1 {
		u.Result.Err = errors.Newf(errors.ErrCodeBadState, "expected 1 item, got %d", len(res.Items))
	} else {
		if err = attributevalue.UnmarshalMapWithOptions(res.Items[0], u.target, u.db.decoderConfigurator); err != nil {
			u.Result.Err = errors.NewUnknownf("failed to unmarshal item, error: %w", err)
		}
	}

	return u.Result, u.Result.Err
}

func applyIncrement(sb *strings.Builder, field string, v increment, parameters *[]any) {
	sb.WriteString(field)
	switch {
	case v.intValue != nil:
		inc := *v.intValue
		if inc == 0 {
			return
		}
		if inc >= 0 {
			sb.WriteString(" + ?")
		} else {
			inc = -inc
			sb.WriteString(" - ?")
		}
		*parameters = append(*parameters, inc)
	case v.uintValue != nil:
		inc := *v.uintValue
		if inc == 0 {
			return
		}
		sb.WriteString(" + ?")
		*parameters = append(*parameters, inc)

	case v.floatValue != nil:
		inc := *v.floatValue
		if inc == 0 {
			return
		}
		if inc >= 0 {
			sb.WriteString(" + ?")
		} else {
			inc = -inc
			sb.WriteString(" - ?")
		}
		*parameters = append(*parameters, inc)
	}
}
