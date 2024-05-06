package tag

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/southernlabs-io/go-fw/errors"
	"github.com/southernlabs-io/go-fw/syncmap"
)

// FieldName returns the name of the field based on the tag names given, the first match will be returned.
// It returns true when the name was found.
func FieldName(field reflect.StructField, tagNames ...string) (string, bool) {
	if len(tagNames) == 0 {
		return "", false
	}

	for _, tagName := range tagNames {
		fieldTag, found := field.Tag.Lookup(tagName)
		if !found {
			continue
		}

		name, _, _ := strings.Cut(fieldTag, ",")
		return name, true
	}

	return "", false
}

var fieldNamesCache = syncmap.New[string, map[string]string]()

// FieldNames returns a map of field names based on the tag names given, the first match will be returned.
// This function results are cached, so:
//   - It is safe to call it concurrently.
//   - It is safe to call it multiple times with the same structType and tagNames.
//   - The returned map should not be modified.
//   - The key for the cache is the structType and tagNames as provided, so changing the tagNames will result in a new cache entry.
func FieldNames(structType reflect.Type, tagNames ...string) (map[string]string, error) {
	if structType.Kind() != reflect.Struct {
		return nil, errors.Newf(errors.ErrCodeBadArgument, "structType must be a struct, given: %s", structType)
	}

	if len(tagNames) == 0 {
		return nil, errors.Newf(errors.ErrCodeBadArgument, "tagNames are required")
	}

	key := fmt.Sprintf("%s:%s", structType, strings.Join(tagNames, ","))
	fieldMap := fieldNamesCache.LoadOrStore(key, func(string) (value map[string]string) {
		return fieldNames(structType, tagNames)
	})

	return fieldMap, nil
}

// fieldNames returns a map of field names based on the tag names given, the first match will be returned.
func fieldNames(structType reflect.Type, tagNames []string) map[string]string {
	resp := map[string]string{}
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		if field.Anonymous {
			embeddedFieldMap := fieldNames(field.Type, tagNames)
			for name, value := range embeddedFieldMap {
				if _, found := resp[name]; !found {
					resp[name] = value
				}
			}
		} else {
			name, found := FieldName(field, tagNames...)
			if !found {
				continue
			}
			if _, found := resp[name]; !found {
				resp[name] = field.Name
			}
		}
	}
	return resp
}
