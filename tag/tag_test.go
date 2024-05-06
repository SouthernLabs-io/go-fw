package tag_test

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/southernlabs-io/go-fw/tag"
)

type StructNoTags struct {
	Field1 string
	Field2 int
}

type StructWithYAMLTagsAndSomePrivate struct {
	privateYAMLField1 string `yaml:"private_yaml_field1,omitempty"`
	privateField2     int
}

func (s StructWithYAMLTagsAndSomePrivate) PrivateYAMLField1() string {
	return s.privateYAMLField1
}

func (s StructWithYAMLTagsAndSomePrivate) PrivateField2() int {
	return s.privateField2
}

type StructWithJSONTags struct {
	JSONField1 string `json:"json_field1"`
	JSONField2 int    `json:"json_field2,omitempty"`
}

type StructWithJSONAndYAMLTags struct {
	JSONYAMLField1 string `json:"json_field1" yaml:"yaml_field1"`
}

type StructAll struct {
	StructNoTags
	StructWithYAMLTagsAndSomePrivate
	StructWithJSONTags
	StructWithJSONAndYAMLTags //nolint:govet
}

func TestFieldNamesFailWithNoStruct(t *testing.T) {
	fieldMap, err := tag.FieldNames(reflect.TypeOf("not a struct"), "json")
	require.Error(t, err)
	require.Nil(t, fieldMap)
}

func TestFieldNamesStructNoTags(t *testing.T) {
	fieldMap, err := tag.FieldNames(reflect.TypeOf(StructNoTags{}), "json")
	require.NoError(t, err)
	require.Empty(t, fieldMap)
}

func TestFieldNamesStructAll(t *testing.T) {
	structType := reflect.TypeOf(StructAll{})

	// It should be empty if we don't look for the right tag
	fieldMap, err := tag.FieldNames(structType, "non-existent-tag")
	require.NoError(t, err)
	require.Empty(t, fieldMap)

	fieldMap, err = tag.FieldNames(structType, "yaml")
	require.NoError(t, err)
	require.Equal(t,
		map[string]string{
			"private_yaml_field1": "privateYAMLField1",
			"yaml_field1":         "JSONYAMLField1",
		},
		fieldMap,
	)

	// It should have json and yaml tags
	fieldMap, err = tag.FieldNames(structType, "yaml", "json", "non-existent-tag")
	require.NoError(t, err)
	require.Equal(t,
		map[string]string{
			"private_yaml_field1": "privateYAMLField1",
			"json_field1":         "JSONField1",
			"json_field2":         "JSONField2",
			"yaml_field1":         "JSONYAMLField1",
		},
		fieldMap,
	)

	// Inverting the tags order should change the result
	fieldMap, err = tag.FieldNames(structType, "json", "yaml", "non-existent-tag")
	require.NoError(t, err)
	require.Equal(t,
		map[string]string{
			"private_yaml_field1": "privateYAMLField1",
			"json_field1":         "JSONField1",
			"json_field2":         "JSONField2",
		},
		fieldMap,
	)
}
