package partiql_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/aws/partiql"
)

func TestPartiQLBuilder_Tuple(t *testing.T) {
	pb := partiql.NewPartiQLBuilder()
	pb.WriteBeginTuple()
	pb.WriteKey("id").WriteString("asd")
	pb.WriteEndTuple()

	require.Equal(t, "{'id':'asd'}", pb.String())
}

func TestPartiQLBuilder_TupleWithTwoKeys(t *testing.T) {
	pb := partiql.NewPartiQLBuilder()
	pb.WriteBeginTuple()
	pb.WriteKey("id").WriteString("asd")
	pb.WriteKey("name").WriteString("John Doe")
	pb.WriteEndTuple()

	require.Equal(t, "{'id':'asd','name':'John Doe'}", pb.String())
}

func TestPartiQLBuilder_Types(t *testing.T) {
	pb := partiql.NewPartiQLBuilder()
	pb.WriteBeginTuple()
	pb.WriteKey("string").WriteString("asd")
	pb.WriteKey("stringArray").WriteStringArray([]string{"a", "b", "c"})
	pb.WriteKey("stringBag").WriteStringBag([]string{"a", "b", "c"})
	pb.WriteKey("integer").WriteInteger(30)
	pb.WriteKey("integerArray").WriteIntegerArray([]int{1, 2, 3})
	pb.WriteKey("integerBag").WriteIntegerBag([]int{1, 2, 3})
	pb.WriteKey("uinteger").WriteUInteger(21)
	pb.WriteKey("uintegerArray").WriteUIntegerArray([]uint{1, 2, 3})
	pb.WriteKey("uintegerBag").WriteUIntegerBag([]uint{1, 2, 3})
	pb.WriteKey("float").WriteFloat(1.8)
	pb.WriteKey("floatArray").WriteFloatArray([]float64{32.5, 34.1, 36.7})
	pb.WriteKey("floatBag").WriteFloatBag([]float64{32.5, 34.1, 36.7})
	pb.WriteKey("boolean").WriteBoolean(true)
	pb.WriteKey("booleanArray").WriteBooleanArray([]bool{true, false, true})
	pb.WriteKey("booleanBag").WriteBooleanBag([]bool{true, false, true})
	pb.WriteKey("nullable").WriteNull()
	pb.WriteEndTuple()

	require.Equal(t,
		"{'string':'asd','stringArray':['a','b','c'],'stringBag':<<'a','b','c'>>,'integer':30,"+
			"'integerArray':[1,2,3],'integerBag':<<1,2,3>>,'uinteger':21,'uintegerArray':[1,2,3],'uintegerBag':<<1,2,3>>,"+
			"'float':1.8,'floatArray':[32.5,34.1,36.7],'floatBag':<<32.5,34.1,36.7>>,'boolean':true,"+
			"'booleanArray':[true,false,true],'booleanBag':<<true,false,true>>,'nullable':NULL}",
		pb.String())
}

func TestPartiQLBuilder_Nested(t *testing.T) {
	pb := partiql.NewPartiQLBuilder()
	pb.WriteBeginTuple()
	pb.WriteKey("id").WriteString("asd")
	pb.WriteKey("nested")
	pb.WriteBeginTuple()
	pb.WriteKey("name").WriteString("John Doe")
	pb.WriteEndTuple()
	pb.WriteEndTuple()

	require.Equal(t, "{'id':'asd','nested':{'name':'John Doe'}}", pb.String())
}

func TestPartiQLBuilder_BadState(t *testing.T) {
	pb := partiql.NewPartiQLBuilder()
	require.Panics(t, func() {
		pb.WriteBeginTuple()
		pb.WriteBeginTuple()
		pb.WriteEndTuple()
		pb.WriteEndTuple()
		pb.WriteEndTuple()
	})
}
