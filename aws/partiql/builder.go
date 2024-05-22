package partiql

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/southernlabs-io/go-fw/errors"
)

const (
	sq  = '\''
	sep = ','
	lcb = '{'
	rcb = '}'
)

type Builder struct {
	sb *strings.Builder

	tupleDepth int
	writeSep   bool
}

func NewPartiQLBuilder() *Builder {
	return &Builder{sb: &strings.Builder{}}
}

func (pb *Builder) WithStringBuilder(sb *strings.Builder) *Builder {
	pb.sb = sb
	return pb
}

func (pb *Builder) Reset() {
	pb.sb.Reset()
	pb.tupleDepth = 0
	pb.writeSep = false
}

func (pb *Builder) writeSepIfNeeded() {
	if pb.writeSep {
		pb.sb.WriteByte(sep)
	} else {
		pb.writeSep = true
	}
}

func (pb *Builder) WriteBeginTuple() {
	pb.writeSepIfNeeded()
	pb.sb.WriteByte(lcb)
	pb.tupleDepth++
	pb.writeSep = false
}

func (pb *Builder) WriteEndTuple() {
	if pb.tupleDepth == 0 {
		panic(errors.Newf(errors.ErrCodeBadState, "unbalanced struct end"))
	}
	pb.sb.WriteByte(rcb)
	pb.tupleDepth--
	pb.writeSep = true
}

func (pb *Builder) WriteKey(name string) *Builder {
	pb.writeSepIfNeeded()
	pb.sb.WriteByte(sq)
	pb.sb.WriteString(name)
	pb.sb.WriteString("':")
	pb.writeSep = false
	return pb
}

func (pb *Builder) WriteString(value string) *Builder {
	pb.writeSepIfNeeded()
	pb.sb.WriteByte(sq)
	pb.sb.WriteString(strings.ReplaceAll(value, "'", "''"))
	pb.sb.WriteByte(sq)
	return pb
}

func (pb *Builder) WriteInteger(value int64) *Builder {
	pb.writeSepIfNeeded()
	pb.sb.WriteString(strconv.FormatInt(value, 10))
	return pb
}

func (pb *Builder) WriteUInteger(value uint64) *Builder {
	pb.writeSepIfNeeded()
	pb.sb.WriteString(strconv.FormatUint(value, 10))
	return pb
}

func (pb *Builder) WriteFloat(value float64) *Builder {
	pb.writeSepIfNeeded()
	pb.sb.WriteString(strconv.FormatFloat(value, 'f', -1, 64))
	return pb
}

func (pb *Builder) WriteBoolean(value bool) *Builder {
	pb.writeSepIfNeeded()
	pb.sb.WriteString(strconv.FormatBool(value))
	return pb
}

func (pb *Builder) WriteBooleanArray(values []bool) *Builder {
	pb.writeArray(values, func(v any) {
		pb.WriteBoolean(v.(bool))
	})
	return pb
}

func (pb *Builder) WriteBooleanBag(values []bool) *Builder {
	pb.writeBag(values, func(v any) {
		pb.WriteBoolean(v.(bool))
	})
	return pb
}

func (pb *Builder) WriteNull() *Builder {
	pb.writeSepIfNeeded()
	pb.sb.WriteString("NULL")
	return pb
}

func (pb *Builder) WriteBeginBag() *Builder {
	pb.writeSepIfNeeded()
	pb.sb.WriteString("<<")
	pb.writeSep = false
	return pb
}

func (pb *Builder) WriteEndBag() *Builder {
	pb.sb.WriteString(">>")
	pb.writeSep = true
	return pb
}

func (pb *Builder) writeBag(values any, writeFunc func(any)) {
	pb.WriteBeginBag()
	slice := reflect.ValueOf(values)
	for i := 0; i < slice.Len(); i++ {
		writeFunc(slice.Index(i).Interface())
	}
	pb.WriteEndBag()
}

func (pb *Builder) WriteStringBag(values []string) *Builder {
	pb.writeBag(values, func(v any) {
		pb.WriteString(v.(string))
	})
	return pb
}

func (pb *Builder) WriteIntegerBag(values []int) *Builder {
	pb.writeBag(values, func(v any) {
		pb.WriteInteger(int64(v.(int)))
	})
	return pb
}

func (pb *Builder) WriteUIntegerBag(values []uint) *Builder {
	pb.writeBag(values, func(v any) {
		pb.WriteUInteger(uint64(v.(uint)))
	})
	return pb
}

func (pb *Builder) WriteFloatBag(values []float64) *Builder {
	pb.writeBag(values, func(v any) {
		pb.WriteFloat(v.(float64))
	})
	return pb
}

func (pb *Builder) WriteBeginArray() *Builder {
	pb.writeSepIfNeeded()
	pb.sb.WriteByte('[')
	pb.writeSep = false
	return pb
}

func (pb *Builder) WriteEndArray() *Builder {
	pb.sb.WriteByte(']')
	pb.writeSep = true
	return pb
}

func (pb *Builder) writeArray(values any, writeFunc func(any)) {
	pb.WriteBeginArray()
	slice := reflect.ValueOf(values)
	for i := 0; i < slice.Len(); i++ {
		writeFunc(slice.Index(i).Interface())
	}
	pb.WriteEndArray()
}

func (pb *Builder) WriteStringArray(values []string) *Builder {
	pb.writeArray(values, func(v any) {
		pb.WriteString(v.(string))
	})
	return pb
}

func (pb *Builder) WriteIntegerArray(values []int) *Builder {
	pb.writeArray(values, func(v any) {
		pb.WriteInteger(int64(v.(int)))
	})
	return pb
}

func (pb *Builder) WriteUIntegerArray(values []uint) *Builder {
	pb.writeArray(values, func(v any) {
		pb.WriteUInteger(uint64(v.(uint)))
	})
	return pb
}

func (pb *Builder) WriteFloatArray(values []float64) *Builder {
	pb.writeArray(values, func(v any) {
		pb.WriteFloat(v.(float64))
	})
	return pb
}

func (pb *Builder) String() string {
	return pb.sb.String()
}
