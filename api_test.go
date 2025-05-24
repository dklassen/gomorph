package gomorph_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/dklassen/gomorph"
	"github.com/stretchr/testify/assert"
)

func equalTypedValue(a, b gomorph.TypedValue) bool {
	return a.Type() == b.Type() && reflect.DeepEqual(a.Value(), b.Value())
}

// Convert int to string or return an error if not converatible
type intToStringConverter struct {
	gomorph.TypeMap[int, string]
}

func (c intToStringConverter) From(v any) (any, error) {
	i, ok := v.(int)
	if !ok {
		return nil, errors.New("expected int")
	}
	return string(rune(i)), nil
}

func (c intToStringConverter) SourceType() reflect.Type { return reflect.TypeOf(0) }
func (c intToStringConverter) TargetType() reflect.Type { return reflect.TypeOf("") }

// alwaysPassValidator will always return the value as is
type alwaysPassValidator struct {
	gomorph.TypeMap[string, string]
}

func (v alwaysPassValidator) From(val any) (any, error) { return val, nil }
func (v alwaysPassValidator) SourceType() reflect.Type  { return reflect.TypeOf("") }
func (v alwaysPassValidator) TargetType() reflect.Type  { return reflect.TypeOf("") }

// Failing validator will always return an error
type failingValidator struct {
	gomorph.TypeMap[int, int]
}

func (f failingValidator) From(val any) (any, error) {
	return nil, errors.New("validation failed")
}
func (f failingValidator) SourceType() reflect.Type { return reflect.TypeOf(0) }
func (f failingValidator) TargetType() reflect.Type { return reflect.TypeOf(0) }

func TestFieldMappingBuilder_NoConverter(t *testing.T) {
	testValue := "noop"
	src := gomorph.NewField[string]("src")
	dst := gomorph.NewField[string]("dst")

	mapping := gomorph.From[string, string](src).
		To(dst).
		SkipConversion().
		ValidateWith(alwaysPassValidator{}).
		Build()
	result, err := mapping.Map(testValue)

	expected := gomorph.NewTypedValue(testValue)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !equalTypedValue(result.MappedValue(), expected) {
		t.Fatalf("expected %v, got %v", expected, result.MappedValue())
	}
}

func TestFieldMappingBuilder_BasicFlow(t *testing.T) {
	src := gomorph.NewField[int]("src")
	dst := gomorph.NewField[string]("dst")

	mapping := gomorph.From[int, string](src).
		To(dst).
		ConvertWith(intToStringConverter{}).
		ValidateWith(alwaysPassValidator{}).
		Build()

	result, err := mapping.Map(65) // ASCII 65 = 'A'

	expected := gomorph.NewTypedValue("A")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equalTypedValue(result.MappedValue(), expected) {
		t.Fatalf("expected 'A', got %v", result.MappedValue())
	}
}

func TestFieldMappingBuilder_ConversionError(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic but got none")
			return
		}
		assert.Contains(t, r.(string), "first mapper must accept string, got int")
	}()

	src := gomorph.NewField[string]("src")
	dst := gomorph.NewField[string]("dst")

	mapping := gomorph.From[string, string](src).
		To(dst).
		ConvertWith(intToStringConverter{}). // intentionally wrong type
		ValidateWith(alwaysPassValidator{}).
		Build()

	mapping.Map("bad-input")
}

func TestFieldMappingBuilder_ValidatorFails(t *testing.T) {
	src := gomorph.NewField[int]("src")
	dst := gomorph.NewField[int]("dst")

	validator := failingValidator{}

	mapping := gomorph.From[int, int](src).
		To(dst).
		SkipConversion().
		ValidateWith(validator).
		Build()

	expectedErr := "mapper chain failed at step 1: validation failed"
	_, err := mapping.Map(42)
	if err == nil || err.Error() != expectedErr {
		t.Fatalf("expected '%s' error, got %v", expectedErr, err)
	}
}
