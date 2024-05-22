package dynamodb

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go/middleware"

	"github.com/southernlabs-io/go-fw/config"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/functional/slices"
	"github.com/southernlabs-io/go-fw/log"
)

var timeType = reflect.TypeOf(time.Time{})

func NewAWSDynamoDBClient(awsConfig aws.Config) *dynamodb.Client {
	return dynamodb.NewFromConfig(awsConfig)
}

type DynamoDB struct {
	client              *dynamodb.Client
	encoderConfigurator func(*attributevalue.EncoderOptions)
	decoderConfigurator func(*attributevalue.DecoderOptions)
	tablePrefix         string

	NowFunc           func() time.Time
	TimeMarshalerFunc func(time.Time) any
}

func NewDynamoDB(conf config.Config, dynamoDBClient *dynamodb.Client) *DynamoDB {
	return &DynamoDB{
		client: dynamoDBClient,
		encoderConfigurator: func(opt *attributevalue.EncoderOptions) {
			opt.TagKey = "partiql"
			opt.EncodeTime = func(t time.Time) (types.AttributeValue, error) {
				// Encode as unix time with micro precision on the decimal part
				value := strconv.FormatFloat(float64(t.UnixMicro())/1e6, 'f', -1, 64)
				return &types.AttributeValueMemberN{
					Value: value,
				}, nil
			}
		},
		decoderConfigurator: func(opt *attributevalue.DecoderOptions) {
			opt.TagKey = "partiql"
			opt.DecodeTime.N = func(n string) (time.Time, error) {
				float, err := strconv.ParseFloat(n, 64)
				if err != nil {
					return time.Time{}, &attributevalue.UnmarshalError{
						Err: err, Value: n, Type: timeType,
					}
				}
				// Decode as unix time with micro precision on the decimal part
				return time.UnixMicro(int64(float * 1e6)), nil
			}
		},
		NowFunc: func() time.Time {
			// Return time with microsecond precision. Postgres timestamp type has microsecond precision.
			return time.UnixMicro(time.Now().UnixMicro())
		},
		TimeMarshalerFunc: func(t time.Time) any {
			return float64(t.UnixMicro()) / 1e6
		},
		tablePrefix: conf.Env.Name + "_",
	}
}

// Model is a model with a primary key
type Model interface {
	GetTableName() string
	GetPK() PrimaryKey
	SetPK(PrimaryKey)
	GetCreateTs() time.Time
	SetCreateTs(time.Time)
}

// PrimaryKey represents a DynamoDB primary key. Provide a RangeKey and Range value if the model has a range key.
type PrimaryKey struct {
	HashKey  string
	Hash     string
	RangeKey string
	Range    any
}

// ModelWithUpdateTs is a model that can be updated
type ModelWithUpdateTs interface {
	GetUpdateTsKey() string
	GetUpdateTs() time.Time
	SetUpdateTs(time.Time)
}

// ExecutionResult represents the result of an execution
type ExecutionResult struct {
	Err              error
	Metadata         middleware.Metadata
	ConsumedCapacity *types.ConsumedCapacity
}

// CreateTable creates a table for the given model. The model PrimaryKey must be set before calling this method.
// The table name will be composed as: Env.Name + "_" + model.GetTableName().
func (d *DynamoDB) CreateTable(ctx context.Context, model Model) error {
	tableName := d.tablePrefix + model.GetTableName()
	log.GetLoggerFromCtx(ctx).Debugf("Creating table: %s", tableName)

	pk := model.GetPK()
	if pk == (PrimaryKey{}) {
		return errors.Newf(errors.ErrCodeBadArgument, "missing primary key for model")
	}
	input := &dynamodb.CreateTableInput{
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String(pk.HashKey),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String(pk.HashKey),
				KeyType:       types.KeyTypeHash,
			},
		},
		TableName:   aws.String(tableName),
		BillingMode: types.BillingModePayPerRequest,
	}

	if pk.RangeKey != "" {
		valueType := reflect.TypeOf(pk.Range)
		var dynamodbAttrType types.ScalarAttributeType
		switch valueType.Kind() {
		case reflect.String:
			dynamodbAttrType = types.ScalarAttributeTypeS
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
			reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
			reflect.Float32, reflect.Float64:
			dynamodbAttrType = types.ScalarAttributeTypeN
		case reflect.Struct:
			// Check if it is time.Time
			if valueType == timeType {
				dynamodbAttrType = types.ScalarAttributeTypeN
			}
		default:
		}
		if dynamodbAttrType == "" {
			return errors.NewUnknownf("unsupported range key type: %s", valueType)
		}
		input.AttributeDefinitions = append(input.AttributeDefinitions, types.AttributeDefinition{
			AttributeName: aws.String(pk.RangeKey),
			AttributeType: dynamodbAttrType,
		})
		input.KeySchema = append(input.KeySchema, types.KeySchemaElement{
			AttributeName: aws.String(pk.RangeKey),
			KeyType:       types.KeyTypeRange,
		})
	}

	_, err := d.client.CreateTable(ctx, input)
	if err != nil {
		return errors.NewUnknownf("failed to create table: %s, error: %w", tableName, err)
	}
	return nil
}

// CreateTables creates tables for the given models. It will call CreateTable for each model, and return an aggregated
// error if any CreateTable call fails.
func (d *DynamoDB) CreateTables(ctx context.Context, models ...Model) error {
	var errs []error
	for _, model := range models {
		err := d.CreateTable(ctx, model)
		if err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.NewUnknownf("Some tables where not created, errors: %w", errors.Join(errs...))
	}
	return nil
}

// DeleteTable deletes a table for the given model.
func (d *DynamoDB) DeleteTable(ctx context.Context, model Model) error {
	tableName := d.tablePrefix + model.GetTableName()
	log.GetLoggerFromCtx(ctx).Debugf("Deleting table: %s", tableName)

	input := &dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	}

	_, err := d.client.DeleteTable(ctx, input)
	if err != nil {
		return errors.NewUnknownf("failed to delete table: %s, error: %w", tableName, err)
	}

	return nil
}

// ListTables returns a list of tables that match the table prefix. The table prefix is: Env.Name + "_".
func (d *DynamoDB) ListTables(ctx context.Context) ([]string, error) {
	log.GetLoggerFromCtx(ctx).Debugf("Listing tables with prefix: %s", d.tablePrefix)
	input := &dynamodb.ListTablesInput{}
	output, err := d.client.ListTables(ctx, input)
	if err != nil {
		return nil, errors.NewUnknownf("failed to list tables, error: %w", err)
	}

	tables := slices.Filter(output.TableNames, func(table string) bool {
		return strings.HasPrefix(table, d.tablePrefix)
	})

	return tables, nil
}

// DeleteAllTables deletes all tables returned by ListTables. It will call DeleteTable for each table, and return an
// aggregated error if any DeleteTable call fails.
func (d *DynamoDB) DeleteAllTables(ctx context.Context) error {
	logger := log.GetLoggerFromCtx(ctx)
	logger.Debugf("Deleting all tables with prefix: %s", d.tablePrefix)
	tables, err := d.ListTables(ctx)
	if err != nil {
		return errors.NewUnknownf("failed to list tables for deletion, error: %w", err)
	}

	var errs []error
	for _, table := range tables {
		input := &dynamodb.DeleteTableInput{
			TableName: aws.String(table),
		}

		logger.Debugf("Deleting table: %s", table)
		_, err = d.client.DeleteTable(ctx, input)
		if err != nil {
			// We use a simple error here to avoid repeating the stack on every join
			errs = append(errs, fmt.Errorf("failed to delete table: %s, error: %w", table, err))
		}
	}
	if len(errs) > 0 {
		return errors.NewUnknownf("Some tables where not deleted, errors: %w", errors.Join(errs...))
	}
	return nil
}
