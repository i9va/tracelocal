package model

import "fmt"

// AttributeKind identifies the concrete type stored in an AttributeValue.
type AttributeKind int

const (
	KindString      AttributeKind = iota
	KindBool
	KindInt64
	KindFloat64
	KindStringSlice
	KindBoolSlice
	KindInt64Slice
	KindFloat64Slice
)

// AttributeValue is a typed, immutable value for an OTel attribute.
// Construct with the typed constructors (StringValue, BoolValue, …).
type AttributeValue struct {
	kind         AttributeKind
	strVal       string
	boolVal      bool
	int64Val     int64
	float64Val   float64
	strSlice     []string
	boolSlice    []bool
	int64Slice   []int64
	float64Slice []float64
}

// Constructors.
func StringValue(v string) AttributeValue       { return AttributeValue{kind: KindString, strVal: v} }
func BoolValue(v bool) AttributeValue           { return AttributeValue{kind: KindBool, boolVal: v} }
func Int64Value(v int64) AttributeValue         { return AttributeValue{kind: KindInt64, int64Val: v} }
func Float64Value(v float64) AttributeValue     { return AttributeValue{kind: KindFloat64, float64Val: v} }
func StringSliceValue(v []string) AttributeValue  { return AttributeValue{kind: KindStringSlice, strSlice: v} }
func BoolSliceValue(v []bool) AttributeValue      { return AttributeValue{kind: KindBoolSlice, boolSlice: v} }
func Int64SliceValue(v []int64) AttributeValue    { return AttributeValue{kind: KindInt64Slice, int64Slice: v} }
func Float64SliceValue(v []float64) AttributeValue { return AttributeValue{kind: KindFloat64Slice, float64Slice: v} }

// Accessors.
func (a AttributeValue) Kind() AttributeKind   { return a.kind }
func (a AttributeValue) AsString() string      { return a.strVal }
func (a AttributeValue) AsBool() bool          { return a.boolVal }
func (a AttributeValue) AsInt64() int64        { return a.int64Val }
func (a AttributeValue) AsFloat64() float64    { return a.float64Val }
func (a AttributeValue) AsStringSlice() []string   { return a.strSlice }
func (a AttributeValue) AsBoolSlice() []bool       { return a.boolSlice }
func (a AttributeValue) AsInt64Slice() []int64     { return a.int64Slice }
func (a AttributeValue) AsFloat64Slice() []float64 { return a.float64Slice }

// String returns a human-readable representation of the value.
func (a AttributeValue) String() string {
	switch a.kind {
	case KindString:
		return a.strVal
	case KindBool:
		return fmt.Sprintf("%t", a.boolVal)
	case KindInt64:
		return fmt.Sprintf("%d", a.int64Val)
	case KindFloat64:
		return fmt.Sprintf("%g", a.float64Val)
	case KindStringSlice:
		return fmt.Sprintf("%v", a.strSlice)
	case KindBoolSlice:
		return fmt.Sprintf("%v", a.boolSlice)
	case KindInt64Slice:
		return fmt.Sprintf("%v", a.int64Slice)
	case KindFloat64Slice:
		return fmt.Sprintf("%v", a.float64Slice)
	default:
		return ""
	}
}

// Attribute is a key-value pair attached to a span, event, link, or resource.
type Attribute struct {
	Key   string
	Value AttributeValue
}

// Attributes is an ordered slice of key-value pairs.
type Attributes []Attribute

// Get returns the value for key and true if found, otherwise the zero value and false.
func (attrs Attributes) Get(key string) (AttributeValue, bool) {
	for _, a := range attrs {
		if a.Key == key {
			return a.Value, true
		}
	}
	return AttributeValue{}, false
}

// GetString returns the string value for key, or "" if absent or not a string.
func (attrs Attributes) GetString(key string) string {
	v, _ := attrs.Get(key)
	return v.AsString()
}
