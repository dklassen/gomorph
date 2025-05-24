package gomorph

import (
	"fmt"
	"reflect"
)

var _ Field = FieldDef[any]{}

// Field represents a logical property of a data record or structure. It describes what a value should look like but does not
// contain any actual data. It serves as a contract for the data that can be stored in a record.
type Field interface {
	Name() string
	Type() reflect.Type
}

type FieldDef[T any] struct {
	name string
	typ  reflect.Type
}

func NewField[T any](name string) FieldDef[T] {
	var zero T

	return FieldDef[T]{
		name: name,
		typ:  reflect.TypeOf(zero),
	}
}

func (f FieldDef[T]) Name() string {
	return f.name
}

func (f FieldDef[T]) Type() reflect.Type {
	return f.typ
}

// TypedValue is a value that has a type.
// It represents a data value and its type.
type TypedValue struct {
	value any
	typ   reflect.Type
}

func NewTypedValue(value any) TypedValue {
	return TypedValue{
		value: value,
		typ:   reflect.TypeOf(value),
	}
}

func (v TypedValue) Value() any {
	return v.value
}

func (v TypedValue) Type() reflect.Type {
	return v.typ
}

func As[T any](v TypedValue) T {
	var zero T
	if v.typ != reflect.TypeOf(zero) {
		panic(fmt.Errorf("value is not of type %T", zero))
	}
	return v.value.(T)
}

func UnwrapAs[T any](fm FieldMappingResult) T {
	var zero T
	if fm.MappedValue().Type() != reflect.TypeOf(zero) {
		panic(fmt.Errorf("value is not of type %T", zero))
	}
	return fm.MappedValue().Value().(T)
}

type FieldMappingResult struct {
	targetField Field
	mappedValue TypedValue
}

func (r FieldMappingResult) TargetField() Field {
	return r.targetField
}

func (r FieldMappingResult) MappedValue() TypedValue {
	return r.mappedValue
}

func NewFieldMappingResult(targetField Field, value TypedValue) FieldMappingResult {
	return FieldMappingResult{
		targetField: targetField,
		mappedValue: value,
	}
}
