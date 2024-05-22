package partiql_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/aws/partiql"
)

type Address struct {
	Street string `json:"street"`
	Number int    `json:"number"`
}

func (a Address) MarshalPartiQL() (any, error) {
	return fmt.Sprintf("%d %s", a.Number, a.Street), nil
}

type Person struct {
	Name        string  `json:"name"`
	Age         int     `json:"age"`
	Addr        Address `json:"addr"`
	PrivateName string  `json:"-"`
	Chan        chan int
}

type Manager struct {
	Person     `partiql:",squash"`
	Department string `partiql:"department"`

	Sub struct {
		Person     `partiql:",squash"`
		Department string `partiql:"department"`
	}

	Scores               []int          `partiql:"scores,bag"`
	SalaryIncreaseFactor []float64      `partiql:"salary_increase_factor"`
	Meta                 map[string]any `partiql:"meta"`
}

func TestPartiQLSerializeStruct(t *testing.T) {
	m := Manager{
		Person: Person{
			Name: "John Doe",
			Age:  30,
			Addr: Address{
				Street: "O'Neil St",
				Number: 123,
			},
		},
		Department: "HR",
		Scores:     []int{1, 2, 3},
		SalaryIncreaseFactor: []float64{
			0.03,
			0.05,
			0.01,
		},
		Sub: struct {
			Person     `partiql:",squash"`
			Department string `partiql:"department"`
		}{
			Person: Person{
				Name: "Mary Smith",
				Age:  25,
				Addr: Address{
					Street: "Second St",
					Number: 321,
				},
			},
			Department: "IT",
		},
		Meta: map[string]any{
			"alt_name": "John",
		},
	}

	enc := partiql.NewEncoder().WithTagNames("partiql", "json").WithTimeMarshaler(func(t time.Time) any {
		return float64(t.UnixMicro()) / 1e6
	})

	partiQLBytes, err := enc.Marshal(m)
	require.NoError(t, err)
	require.NotEmpty(t, partiQLBytes)
	require.Equal(t,
		"{'name':'John Doe','age':30,'addr':'123 O''Neil St','department':'HR',"+
			"'Sub':{'name':'Mary Smith','age':25,'addr':'321 Second St','department':'IT'},'scores':<<1,2,3>>,"+
			"'salary_increase_factor':[0.03,0.05,0.01],'meta':{'alt_name':'John'}}",
		string(partiQLBytes),
	)

}
