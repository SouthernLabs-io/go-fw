package dynamodb_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/aws/dynamodb"
	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/test"
)

func TestDynamoDB(t *testing.T) {
	var ddb *dynamodb.DynamoDB
	var ctx context.Context
	test.FxIntegration(t, test.ModuleTestAWSLocalStackConfig, test.ModuleTestDynamoDB).
		Populate(
			&ctx,
			&ddb,
		)

	person := &Person{Base: Base{ID: "123456"}}
	err := ddb.CreateTable(ctx, person)
	require.NoError(t, err)

	gotPerson := &Person{
		Base: Base{
			ID: "123456",
		},
	}
	_, err = ddb.Get(gotPerson).Execute(ctx)
	require.Error(t, err)
	var fwErr *errors.Error
	require.ErrorAs(t, err, &fwErr)
	require.Equal(t, errors.ErrCodeNotFound, fwErr.Code, fwErr.Error())

	person.Name = "John Doe"
	_, err = ddb.Insert(person).Execute(ctx)
	require.NoError(t, err)

	_, err = ddb.Get(gotPerson).WithConsistentRead().Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, person, gotPerson)

	// Insert again should fail
	_, err = ddb.Insert(person).Execute(ctx)
	require.Error(t, err)
	require.ErrorAs(t, err, &fwErr)
	require.Equal(t, errors.ErrCodeConflict, fwErr.Code, fwErr.Error())

	// Update
	updatedPerson := &Person{
		Base: Base{
			ID: person.ID,
		},
	}
	_, err = ddb.Update(updatedPerson).Set("name", "John Doe Jr.").Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, "John Doe Jr.", updatedPerson.Name)
	require.Equal(t, person.CreateTs, updatedPerson.CreateTs)
	require.Greater(t, updatedPerson.UpdateTs, person.UpdateTs)

	// Insert another person
	person2 := &Person{
		Base: Base{
			ID: "654321",
		},
		Name: "Jane Doe",
	}
	_, err = ddb.Insert(person2).Execute(ctx)
	require.NoError(t, err)

	// Get many
	people := []Person{{Base: Base{ID: "123456"}}, {Base: Base{ID: "654321"}}}
	_, err = ddb.Query(&people).Execute(ctx)
	require.NoError(t, err)
	require.Len(t, people, 2)
	require.Contains(t, people, *updatedPerson)
	require.Contains(t, people, *person2)

	// Delete
	_, err = ddb.Delete(person).Execute(ctx)
	require.NoError(t, err)

	// Delete again should fail
	_, err = ddb.Delete(person).Execute(ctx)
	require.Error(t, err)
	require.ErrorAs(t, err, &fwErr)
	require.Equal(t, errors.ErrCodeNotFound, fwErr.Code)

	// Get again should fail
	_, err = ddb.Get(gotPerson).Execute(ctx)
	require.Error(t, err)
	require.ErrorAs(t, err, &fwErr)
	require.Equal(t, errors.ErrCodeNotFound, fwErr.Code)
}

func TestDynamoDB_ScanNotAllowed(t *testing.T) {
	var ddb *dynamodb.DynamoDB
	var ctx context.Context
	test.FxIntegration(t, test.ModuleTestAWSLocalStackConfig, test.ModuleTestDynamoDB).
		Populate(
			&ctx,
			&ddb,
		)

	err := ddb.CreateTable(ctx, &Person{Base: Base{ID: "123456"}})
	require.NoError(t, err)

	person := &Person{
		Base: Base{
			ID: "123456",
		},
	}
	_, err = ddb.Get(person).Where("name = ?", "John Doe").Execute(ctx)
	require.Error(t, err)
	var fwErr *errors.Error
	require.ErrorAs(t, err, &fwErr)
	require.Equal(t, errors.ErrCodeBadArgument, fwErr.Code, fwErr.Error())

	_, err = ddb.Query(&[]Person{}).Where("name = ?", "John Doe").Execute(ctx)
	require.Error(t, err)
	require.ErrorAs(t, err, &fwErr)
	require.Equal(t, errors.ErrCodeBadArgument, fwErr.Code, fwErr.Error())

	// Allow scan
	_, err = ddb.Query(&[]Person{}).WithAllowScan().Execute(ctx)
	require.NoError(t, err)
}

func TestDynamoDB_Update(t *testing.T) {
	var ddb *dynamodb.DynamoDB
	var ctx context.Context
	test.FxIntegration(t, test.ModuleTestAWSLocalStackConfig, test.ModuleTestDynamoDB).
		Populate(
			&ctx,
			&ddb,
		)

	err := ddb.CreateTable(ctx, &Person{Base: Base{ID: "123456"}})
	require.NoError(t, err)

	person := &Person{
		Base: Base{
			ID: "123456",
		},
		Name:        "John Doe",
		Age:         30,
		CreditScore: 6.5,
		Money:       -1000,
		Addr: Address{
			Street: "O'Neil St",
			Number: 123,
		},
		Pets:    []string{},
		Friends: []string{},
	}

	// Try delete table first
	_ = ddb.DeleteTable(ctx, person)
	t.Cleanup(func() {
		if !t.Failed() {
			_ = ddb.DeleteTable(ctx, person)
		}
	})
	err = ddb.CreateTable(ctx, person)
	require.NoError(t, err)

	// Insert
	_, err = ddb.Insert(person).Execute(ctx)
	require.NoError(t, err)

	updatedPerson := &Person{
		Base: Base{
			ID: person.ID,
		},
	}

	// Update
	_, err = ddb.Update(updatedPerson).
		Set("name", "John Doe Jr.").
		AddToSet("pets", "peper", "ruf").
		AppendToList("friends", "jane").
		Increment("age", 1).
		Increment("money", -10).
		Increment("credit_score", 0.3).
		Remove("addr").
		Set("meta", map[string]any{"alt_name": "mister dude"}).
		Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, "John Doe Jr.", updatedPerson.Name)
	require.EqualValues(t, 31, updatedPerson.Age)
	require.EqualValues(t, 6.8, updatedPerson.CreditScore)
	require.EqualValues(t, -1010, updatedPerson.Money)
	require.Equal(t, []string{"peper", "ruf"}, updatedPerson.Pets)
	require.Equal(t, []string{"jane"}, updatedPerson.Friends)
	require.Equal(t, Address{}, updatedPerson.Addr)
	require.Equal(t, "mister dude", updatedPerson.Meta["alt_name"])

	// Execute the opposite update
	updatedPerson = &Person{
		Base: Base{
			ID: person.ID,
		},
	}
	_, err = ddb.Update(updatedPerson).
		Set("name", "John Doe").
		RemoveFromSet("pets", "ruf").
		RemoveFromList("friends", 0).
		Increment("age", -1).
		Increment("money", uint(10)).
		Increment("credit_score", -0.3).
		Set("addr", Address{Street: "O'Neil St", Number: 123}).
		Remove("meta").
		Execute(ctx)
	require.NoError(t, err)
	require.Equal(t, "John Doe", updatedPerson.Name)
	require.EqualValues(t, 30, updatedPerson.Age)
	require.EqualValues(t, 6.5, updatedPerson.CreditScore)
	require.EqualValues(t, -1000, updatedPerson.Money)
	require.Equal(t, []string{"peper"}, updatedPerson.Pets)
	require.Equal(t, []string{}, updatedPerson.Friends)
	require.Equal(t, Address{Street: "O'Neil St", Number: 123}, updatedPerson.Addr)
	require.Empty(t, updatedPerson.Meta)
}

func TestDynamoDB_PKWithRange(t *testing.T) {
	var ddb *dynamodb.DynamoDB
	var ctx context.Context
	test.FxIntegration(t, test.ModuleTestAWSLocalStackConfig, test.ModuleTestDynamoDB).
		Populate(
			&ctx,
			&ddb,
		)

	err := ddb.CreateTables(ctx, &Person{Base: Base{ID: "123456"}}, &Building{Base: Base{ID: "123456"}})
	require.NoError(t, err)

	var building Building
	_, err = ddb.Insert(&building).Execute(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, building.GetPK().Hash)
	require.NotEmpty(t, building.GetPK().Range)
}

type Base struct {
	ID       string    `partiql:"id"`
	CreateTs time.Time `partiql:"create_ts"`
	UpdateTs time.Time `partiql:"update_ts"`
}

func (b *Base) SetPK(pk dynamodb.PrimaryKey) {
	b.ID = pk.Hash
}

func (b *Base) GetPK() dynamodb.PrimaryKey {
	return dynamodb.PrimaryKey{HashKey: "id", Hash: b.ID}
}

func (b *Base) SetCreateTs(t time.Time) {
	b.CreateTs = t
}

func (b *Base) GetCreateTs() time.Time {
	return b.CreateTs
}

func (b *Base) SetUpdateTs(t time.Time) {
	b.UpdateTs = t
}

func (b *Base) GetUpdateTs() time.Time {
	return b.UpdateTs
}

func (b *Base) GetUpdateTsKey() string {
	return "update_ts"
}

type Person struct {
	Base        `partiql:",squash"`
	Name        string         `partiql:"name"`
	Age         uint           `partiql:"age"`
	CreditScore float64        `partiql:"credit_score"`
	Money       int            `partiql:"money"`
	Addr        Address        `partiql:"addr"`
	Pets        []string       `partiql:"pets,bag"`
	Friends     []string       `partiql:"friends"`
	Meta        map[string]any `partiql:"meta"`
}

func (p *Person) GetTableName() string {
	return "person"
}

type Address struct {
	Street string `partiql:"street"`
	Number int    `partiql:"number"`
}

type Building struct {
	Base `partiql:",squash"`
	Name string  `partiql:"name"`
	Addr Address `partiql:"addr"`
}

func (b *Building) GetTableName() string {
	return "building"
}

func (b *Building) GetPK() dynamodb.PrimaryKey {
	pk := b.Base.GetPK()
	pk.RangeKey = "create_ts"
	pk.Range = b.CreateTs
	return pk
}
