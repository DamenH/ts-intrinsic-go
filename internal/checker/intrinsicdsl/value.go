// Package intrinsicdsl implements a small JS-like interpreter for
// Intrinsic<Fun, Args> type-level computation.
package intrinsicdsl

import (
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/microsoft/typescript-go/internal/collections"
	"github.com/microsoft/typescript-go/internal/jsnum"
)

type ValueKind int

const (
	KindNumber ValueKind = iota
	KindString
	KindBoolean
	KindNull
	KindUndefined
	KindTypeSentinel // Str holds: "number", "string", "boolean", "never", "unknown", "void"
	KindTuple
	KindObject
	KindFunction
)

type Value struct {
	Kind  ValueKind
	Num   float64
	Str   string
	Bool  bool
	Elems []Value
	Props *collections.OrderedMap[string, Value]
	Fn    *Closure
}

type Closure struct {
	Params []string
	Body   Node
	Env    *Env
}

func NewOrderedMap() *collections.OrderedMap[string, Value] {
	return &collections.OrderedMap[string, Value]{}
}

func NumVal(n float64) Value { return Value{Kind: KindNumber, Num: n} }
func StrVal(s string) Value  { return Value{Kind: KindString, Str: s} }
func BoolVal(b bool) Value   { return Value{Kind: KindBoolean, Bool: b} }

func TupleVal(elems ...Value) Value {
	if elems == nil {
		elems = []Value{}
	}
	return Value{Kind: KindTuple, Elems: elems}
}

func ObjectVal(m *collections.OrderedMap[string, Value]) Value {
	return Value{Kind: KindObject, Props: m}
}

var (
	Null        = Value{Kind: KindNull}
	Undefined   = Value{Kind: KindUndefined}
	NeverType   = Value{Kind: KindTypeSentinel, Str: "never"}
	UnknownType = Value{Kind: KindTypeSentinel, Str: "unknown"}
	VoidType    = Value{Kind: KindTypeSentinel, Str: "void"}
	NumberType  = Value{Kind: KindTypeSentinel, Str: "number"}
	StringType  = Value{Kind: KindTypeSentinel, Str: "string"}
	BooleanType = Value{Kind: KindTypeSentinel, Str: "boolean"}
)

// IsTruthy follows JS truthiness rules.
func IsTruthy(v Value) bool {
	switch v.Kind {
	case KindBoolean:
		return v.Bool
	case KindNumber:
		return v.Num != 0 && !math.IsNaN(v.Num)
	case KindString:
		return v.Str != ""
	case KindNull, KindUndefined:
		return false
	case KindTypeSentinel:
		return v.Str != "never"
	default:
		return true
	}
}

func ValuesEqual(a, b Value) bool {
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case KindNumber:
		return a.Num == b.Num
	case KindString:
		return a.Str == b.Str
	case KindBoolean:
		return a.Bool == b.Bool
	case KindNull, KindUndefined:
		return true
	case KindTypeSentinel:
		return a.Str == b.Str
	case KindTuple:
		if len(a.Elems) != len(b.Elems) {
			return false
		}
		for i := range a.Elems {
			if !ValuesEqual(a.Elems[i], b.Elems[i]) {
				return false
			}
		}
		return true
	case KindObject:
		if a.Props.Size() != b.Props.Size() {
			return false
		}
		for k, av := range a.Props.Entries() {
			bv, ok := b.Props.Get(k)
			if !ok || !ValuesEqual(av, bv) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func TypeofValue(v Value) string {
	switch v.Kind {
	case KindNumber:
		return "number"
	case KindString:
		return "string"
	case KindBoolean:
		return "boolean"
	case KindNull:
		return "null"
	case KindUndefined:
		return "undefined"
	case KindTuple:
		return "tuple"
	case KindObject:
		return "object"
	case KindFunction:
		return "function"
	case KindTypeSentinel:
		return v.Str
	default:
		return "unknown"
	}
}

func CacheKey(v Value) string {
	var b strings.Builder
	writeCacheKey(&b, v)
	return b.String()
}

func writeCacheKey(b *strings.Builder, v Value) {
	switch v.Kind {
	case KindNumber:
		b.WriteByte('n')
		b.WriteString(formatNum(v.Num))
	case KindString:
		b.WriteByte('s')
		b.WriteString(strconv.Itoa(len(v.Str)))
		b.WriteByte(':')
		b.WriteString(v.Str)
	case KindBoolean:
		if v.Bool {
			b.WriteByte('T')
		} else {
			b.WriteByte('F')
		}
	case KindNull:
		b.WriteString("null")
	case KindUndefined:
		b.WriteString("undef")
	case KindTypeSentinel:
		b.WriteString("T:")
		b.WriteString(v.Str)
	case KindTuple:
		b.WriteByte('[')
		for i, e := range v.Elems {
			if i > 0 {
				b.WriteByte(',')
			}
			writeCacheKey(b, e)
		}
		b.WriteByte(']')
	case KindObject:
		b.WriteByte('{')
		first := true
		for k, val := range v.Props.Entries() {
			if !first {
				b.WriteByte(',')
			}
			first = false
			b.WriteString(k)
			b.WriteByte(':')
			writeCacheKey(b, val)
		}
		b.WriteByte('}')
	case KindFunction:
		b.WriteString("fn")
	}
}

func formatNum(n float64) string {
	return jsnum.Number(n).String()
}

func DeepCopy(v Value) Value {
	switch v.Kind {
	case KindTuple:
		elems := make([]Value, len(v.Elems))
		for i, e := range v.Elems {
			elems[i] = DeepCopy(e)
		}
		return TupleVal(elems...)
	case KindObject:
		m := NewOrderedMap()
		for k, val := range v.Props.Entries() {
			m.Set(k, DeepCopy(val))
		}
		return ObjectVal(m)
	default:
		return v
	}
}

func Display(v Value) string {
	var b strings.Builder
	writeDisplay(&b, v)
	return b.String()
}

func writeDisplay(b *strings.Builder, v Value) {
	switch v.Kind {
	case KindNumber:
		b.WriteString(formatNum(v.Num))
	case KindString:
		b.WriteByte('"')
		b.WriteString(v.Str)
		b.WriteByte('"')
	case KindBoolean:
		if v.Bool {
			b.WriteString("true")
		} else {
			b.WriteString("false")
		}
	case KindNull:
		b.WriteString("null")
	case KindUndefined:
		b.WriteString("undefined")
	case KindTypeSentinel:
		b.WriteString(v.Str)
	case KindTuple:
		b.WriteByte('[')
		for i, e := range v.Elems {
			if i > 0 {
				b.WriteString(", ")
			}
			writeDisplay(b, e)
		}
		b.WriteByte(']')
	case KindObject:
		b.WriteByte('{')
		first := true
		for k, val := range v.Props.Entries() {
			if !first {
				b.WriteString(", ")
			}
			first = false
			b.WriteString(k)
			b.WriteString(": ")
			writeDisplay(b, val)
		}
		b.WriteByte('}')
	case KindFunction:
		b.WriteString("<function>")
	}
}

func CompareValues(a, b Value) int {
	if a.Kind == KindNumber && b.Kind == KindNumber {
		switch {
		case a.Num < b.Num:
			return -1
		case a.Num > b.Num:
			return 1
		default:
			return 0
		}
	}
	if a.Kind == KindString && b.Kind == KindString {
		return strings.Compare(a.Str, b.Str)
	}
	return 0
}

func SortValues(vals []Value) {
	sort.SliceStable(vals, func(i, j int) bool {
		return CompareValues(vals[i], vals[j]) < 0
	})
}
