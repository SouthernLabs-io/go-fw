package partiql

import (
	"reflect"
	"time"

	"github.com/southernlabs-io/go-fw/errors"
)

var timeType = reflect.TypeOf(time.Time{})

// Marshaler is the interface implemented by objects that can marshal themselves into PartiQL.
type Marshaler interface {
	MarshalPartiQL() (any, error)
}

// TimeMarshaler is a function that can be used to encode time.Time values into another go value to be converted into PartiQL.
// Check the DefaultTimeMarshaler for an example.
type TimeMarshaler func(t time.Time) any

// DefaultTimeMarshaler converts a time.Time into a string using the RFC3339Nano format.
var DefaultTimeMarshaler = func(t time.Time) any {
	return t.Format(time.RFC3339Nano)
}

// Marshal returns the PartiQL encoding of v using a default encoder.
func Marshal(value any) ([]byte, error) {
	return NewEncoder().Marshal(value)
}

// Encoder is a PartiQL encoder that can be configured.
type Encoder struct {
	b             *Builder
	timeMarshaler TimeMarshaler
	tagNames      []string
}

// NewEncoder returns a new PartiQL encoder with default settings.
func NewEncoder() *Encoder {
	return &Encoder{
		b:             NewPartiQLBuilder(),
		timeMarshaler: DefaultTimeMarshaler,
		tagNames:      []string{"partiql", "dynamodbav", "json", "yaml"},
	}
}

// WithTimeMarshaler sets the time marshaler to be used by the encoder.
func (e *Encoder) WithTimeMarshaler(marshaler TimeMarshaler) *Encoder {
	e.timeMarshaler = marshaler
	return e
}

// WithTagNames sets the tag names to be used by the encoder. It must contain at least one tag name.
func (e *Encoder) WithTagNames(tagNames ...string) *Encoder {
	if len(tagNames) == 0 {
		panic(errors.Newf(errors.ErrCodeBadArgument, "at least one tag name must be provided"))
	}
	e.tagNames = tagNames
	return e
}

// Marshal returns the PartiQL encoding of value.
func (e *Encoder) Marshal(value any) ([]byte, error) {
	e.b.Reset()
	err := marshal(e, reflect.ValueOf(value), tagValue{})
	if err != nil {
		return nil, err
	}
	return []byte(e.b.String()), nil
}

// MarshalCollection returns the PartiQL encoding of value as a collection.
// It will force the value to be encoded as a Bag if asBag is true.
func (e *Encoder) MarshalCollection(value any, asBag bool) ([]byte, error) {
	e.b.Reset()
	err := marshal(e, reflect.ValueOf(value), tagValue{Bag: asBag})
	if err != nil {
		return nil, err
	}
	return []byte(e.b.String()), nil
}

func marshalStruct(enc *Encoder, v reflect.Value, parentTag tagValue) error {
	if !parentTag.Squash {
		enc.b.WriteBeginTuple()
	}
	for _, ft := range reflect.VisibleFields(v.Type()) {
		// Do not marshal unexported fields. We process embedded fields by walking the parent struct.
		if !ft.IsExported() || len(ft.Index) > 1 {
			continue
		}
		var tag tagValue
		for _, tagName := range enc.tagNames {
			tagStr, present := ft.Tag.Lookup(tagName)
			if present {
				tag = parseTag(tagStr)
				break
			}
		}

		if tag.Ignore {
			continue
		}

		fv := v.FieldByIndex(ft.Index)

		// Ignore not supported types and nulls
		switch fv.Kind() {
		case reflect.Invalid, reflect.Chan, reflect.Func, reflect.UnsafePointer, reflect.Uintptr, reflect.Complex64, reflect.Complex128:
			continue
		case reflect.Ptr, reflect.Interface, reflect.Slice, reflect.Array, reflect.Map:
			if fv.IsNil() {
				continue
			}
		default:
		}

		// Ignore empty bags
		if tag.Bag && isEmptyCollection(fv) {
			continue
		}

		if !tag.Squash {
			enc.b.WriteKey(fieldName(ft, tag))
		}
		err := marshal(enc, fv, tag)
		if err != nil {
			return err
		}
	}
	if !parentTag.Squash {
		enc.b.WriteEndTuple()
	}
	return nil
}

func fieldName(ft reflect.StructField, tag tagValue) string {
	if tag.Name != "" {
		return tag.Name
	} else {
		return ft.Name
	}
}

func marshalCollection(enc *Encoder, v reflect.Value, parentTag tagValue) error {
	if parentTag.Bag {
		enc.b.WriteBeginBag()
	} else {
		enc.b.WriteBeginArray()
	}
	for i := 0; i < v.Len(); i++ {
		err := marshal(enc, v.Index(i), tagValue{})
		if err != nil {
			return err
		}
	}
	if parentTag.Bag {
		enc.b.WriteEndBag()
	} else {
		enc.b.WriteEndArray()
	}
	return nil
}

func isEmptyCollection(v reflect.Value) bool {
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice || v.Kind() == reflect.Array {
		return false
	}

	return v.Len() == 0
}

func marshalMap(enc *Encoder, v reflect.Value, _ tagValue) error {
	enc.b.WriteBeginTuple()
	for _, key := range v.MapKeys() {
		enc.b.WriteKey(key.String())
		err := marshal(enc, v.MapIndex(key), tagValue{})
		if err != nil {
			return err
		}
	}
	enc.b.WriteEndTuple()
	return nil
}

func marshal(enc *Encoder, v reflect.Value, parentTag tagValue) error {
	if m, is := v.Interface().(Marshaler); is {
		data, err := m.MarshalPartiQL()
		if err != nil {
			return err
		}
		return marshal(enc, reflect.ValueOf(data), parentTag)
	}

	switch v.Kind() {
	case reflect.Invalid, reflect.Chan, reflect.Func, reflect.UnsafePointer, reflect.Uintptr, reflect.Complex64, reflect.Complex128:
		return nil
	case reflect.Pointer, reflect.Interface:
		return marshal(enc, v.Elem(), parentTag)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		enc.b.WriteInteger(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		enc.b.WriteUInteger(v.Uint())
	case reflect.Float32, reflect.Float64:
		enc.b.WriteFloat(v.Float())
	case reflect.Bool:
		enc.b.WriteBoolean(v.Bool())
	case reflect.String:
		enc.b.WriteString(v.String())
	case reflect.Slice, reflect.Array:
		return marshalCollection(enc, v, parentTag)
	case reflect.Map:
		return marshalMap(enc, v, parentTag)
	case reflect.Struct:
		if v.Type() == timeType {
			encTime := enc.timeMarshaler(v.Interface().(time.Time))
			return marshal(enc, reflect.ValueOf(encTime), parentTag)
		}
		return marshalStruct(enc, v, parentTag)
	default:
		return errors.Newf(errors.ErrCodeBadState, "unknown type: %s, please implement it!", v.Kind())
	}
	return nil
}
