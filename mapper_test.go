package gomorph_test

import (
	"fmt"
	"testing"

	"github.com/dklassen/gomorph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
)

type Input struct {
	InputString string
	InputInt    int
}

type Output struct {
	MappedInputString string
	MappedInputInt    int
}

var (
	CountryCodeField = gomorph.NewField[string]("country_code")
	PercentageField  = gomorph.NewField[float64]("percentage")
)

type StringToIntMapper struct {
	gomorph.TypeMap[string, int]
}

func (m StringToIntMapper) From(source any) (any, error) {
	str, ok := source.(string)
	if !ok {
		return 0, fmt.Errorf("expected string, got %T", source)
	}
	return len(str), nil
}

// IntDoubler doubles an integer
type IntDoubler struct {
	gomorph.TypeMap[int, int]
}

func (m IntDoubler) From(source any) (any, error) {
	i, ok := source.(int)
	if !ok {
		return 0, fmt.Errorf("expected int, got %T", source)
	}
	return i * 2, nil
}

type AlwaysFailingMapper struct {
	gomorph.TypeMap[string, string]
}

func (m AlwaysFailingMapper) From(source any) (any, error) {
	return nil, fmt.Errorf("always fails error value")
}

func TestGetField(t *testing.T) {
	record := gomorph.Record{
		"country_code": "US",
		"percentage":   50.0,
	}

	// Test string field
	t.Run("get string field", func(t *testing.T) {
		val, err := gomorph.GetField(record, CountryCodeField)
		if err != nil {
			t.Errorf("GetField() error = %v", err)
		}
		if val != "US" {
			t.Errorf("GetField() = %v, want %v", val, "US")
		}
	})

	// Test float field
	t.Run("get float field", func(t *testing.T) {
		val, err := gomorph.GetField(record, PercentageField)
		if err != nil {
			t.Errorf("GetField() error = %v", err)
		}
		if val != 50.0 {
			t.Errorf("GetField() = %v, want %v", val, 50.0)
		}
	})

	// Test missing field
	t.Run("missing field", func(t *testing.T) {
		missingField := gomorph.NewField[string]("missing")
		_, err := gomorph.GetField(record, missingField)
		if err == nil {
			t.Errorf("GetField() field %q not found", missingField.Name())
		}
	})
}

func TestChainedMapperTransformations(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		gomorphs []gomorph.TypedMapper
		expected gomorph.TypedValue
		wantErr  bool
	}{
		{
			name:  "should take string length then double it as output",
			input: "hello",
			gomorphs: []gomorph.TypedMapper{
				StringToIntMapper{}, // string -> int
				IntDoubler{},        // int -> int
			},
			expected: gomorph.NewTypedValue(10),
			wantErr:  false,
		},
		{
			name:  "should map a string to an int",
			input: "test",
			gomorphs: []gomorph.TypedMapper{
				StringToIntMapper{}, // string -> int
			},
			expected: gomorph.NewTypedValue(4),
			wantErr:  false,
		},
		{
			name:  "Should take string length then double it twice as output",
			input: "hello",
			gomorphs: []gomorph.TypedMapper{
				StringToIntMapper{}, // string -> int
				IntDoubler{},        // int -> int
				IntDoubler{},        // int -> int
			},
			expected: gomorph.NewTypedValue(20),
			wantErr:  false,
		},
		{
			name:     "Should error on empty chain",
			input:    "test",
			gomorphs: []gomorph.TypedMapper{},
			expected: gomorph.NewTypedValue("test"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chainedMapper := gomorph.NewChainedMapper[string, int](tt.gomorphs...)

			targetField := gomorph.NewField[int]("test")
			mapping := gomorph.NewFieldMapping(
				gomorph.NewField[string]("test"),
				targetField,
				chainedMapper,
			)

			result, err := mapping.Map(tt.input)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result.MappedValue())
			assert.Equal(t, targetField, result.TargetField())
		})
	}
}

func TestChainedMapperConstruction(t *testing.T) {
	tests := []struct {
		name          string
		gomorphs      []gomorph.TypedMapper
		expectedPanic string
	}{
		{
			name: "should panic at incompatible types between gomorphs",
			gomorphs: []gomorph.TypedMapper{
				IntDoubler{},        // int -> int
				StringToIntMapper{}, // string -> int (wrong order)
			},
			expectedPanic: "first mapper must accept string, got int",
		},
		{
			name: "should panic when first gomorph wrong input type",
			gomorphs: []gomorph.TypedMapper{
				IntDoubler{}, // int -> int, but chain expects string -> int
			},
			expectedPanic: "first mapper must accept string, got int",
		},
		{
			name: "should panic when last gomorph wrong output type",
			gomorphs: []gomorph.TypedMapper{
				StringToIntMapper{},
				StringToIntMapper{}, // outputs int when chain expects string
			},
			expectedPanic: "last mapper must produce string, got int",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if r == nil {
					t.Error("expected panic but got none")
					return
				}
				assert.Contains(t, r.(string), tt.expectedPanic)
			}()

			// This should panic
			_ = gomorph.NewChainedMapper[string, string](tt.gomorphs...)
		})
	}
}

func TestChainableMapperFails(t *testing.T) {
	input := "hello"

	sourceField := gomorph.NewField[string]("test")
	targetField := gomorph.NewField[string]("test")

	fieldMapping := gomorph.NewFieldMapping(
		sourceField,
		targetField,
		gomorph.NewChainedMapper[string, string](
			AlwaysFailingMapper{},
		),
	)

	_, err := fieldMapping.Map(input)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mapper chain failed at step 1: always fails error value")
}

// MockTypedMapper is a mock implementation of TypedMapper for testing.
type MockTypedMapper struct{}

func (m MockTypedMapper) SourceType() reflect.Type {
	var zero string
	return reflect.TypeOf(zero)
}

func (m MockTypedMapper) TargetType() reflect.Type {
	var zero int
	return reflect.TypeOf(zero)
}

func (m MockTypedMapper) From(source any) (any, error) {
	str, ok := source.(string)
	if !ok {
		return nil, fmt.Errorf("expected string, got %T", source)
	}
	return len(str), nil
}

func TestSliceMapper_Success(t *testing.T) {
	elementMapper := MockTypedMapper{}
	sliceMapper := gomorph.NewSliceMapper[[]string, []int](elementMapper)

	input := []string{"hello", "world", "test"}
	expected := []int{5, 5, 4}

	result, err := sliceMapper.From(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output, ok := result.([]int)
	if !ok {
		t.Fatalf("expected []int, got %T", result)
	}

	if !reflect.DeepEqual(output, expected) {
		t.Errorf("expected %v, got %v", expected, output)
	}
}

func TestSliceMapper_InvalidSourceType(t *testing.T) {
	elementMapper := MockTypedMapper{}
	sliceMapper := gomorph.NewSliceMapper[[]string, []int](elementMapper)

	input := []int{1, 2, 3} // Invalid input type

	_, err := sliceMapper.From(input)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	expectedErr := "invalid source type: expected []string, got []int"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestSliceMapper_ElementTransformationError(t *testing.T) {
	elementMapper := MockTypedMapper{}
	sliceMapper := gomorph.NewSliceMapper[[]string, []int](elementMapper)

	input := []any{"hello", 123, "test"} // Invalid element type in the slice

	_, err := sliceMapper.From(input)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}

	expectedErr := "invalid source type: expected []string, got []interface {}"
	if err.Error() != expectedErr {
		t.Errorf("expected error %q, got %q", expectedErr, err.Error())
	}
}
func TestStructMapper_MapExportedFields(t *testing.T) {
	fieldMappings := []gomorph.FieldMapper{
		gomorph.NewFieldMapping(
			gomorph.NewField[string]("InputString"),
			gomorph.NewField[string]("MappedInputString"),
			gomorph.NewChainedMapper[string, string](
				gomorph.IdentityMapper[string]{},
			),
		),
		gomorph.NewFieldMapping(
			gomorph.NewField[int]("InputInt"),
			gomorph.NewField[int]("MappedInputInt"),
			gomorph.NewChainedMapper[int, int](
				gomorph.IdentityMapper[int]{},
			),
		),
	}

	outputMapper := gomorph.NewStructMapper[Input, Output](fieldMappings)

	t.Run("basic mapping through exported fields", func(t *testing.T) {
		targetString := "Once upon a time...."
		targetInt := 10

		subject := Input{
			InputString: targetString,
			InputInt:    targetInt,
		}

		result, err := outputMapper.Map(subject)
		require.NoError(t, err)
		require.Equal(t, targetString, result.MappedInputString)
		require.Equal(t, targetInt, result.MappedInputInt)
	})
}

type ComplexInput struct {
	unexportedStringField string
}

func (i *ComplexInput) GetInputString() string {
	return i.unexportedStringField
}
func (i *ComplexInput) SetInputString(s string) {
	i.unexportedStringField = s
}

type ComplexOutput struct {
	mappedInputString string
}

func (o *ComplexOutput) SetMappedInputString(s string) {
	o.mappedInputString = s
}

func TestStructMapper_MapSettersGetters(t *testing.T) {
	getInputStringField := gomorph.NewField[string]("GetInputString")
	setMappedInputStringField := gomorph.NewField[string]("SetMappedInputString")

	fieldMappings := []gomorph.FieldMapper{
		gomorph.From[string, string](getInputStringField).
			To(setMappedInputStringField).
			ConvertWith(gomorph.IdentityMapper[string]{}).
			SkipValidation().
			Build(),
	}

	outputMapper := gomorph.NewStructMapper[ComplexInput, ComplexOutput](fieldMappings)

	t.Run("complex mapping via setters and getters not fields", func(t *testing.T) {
		subject := ComplexInput{
			unexportedStringField: "Foo",
		}

		result, err := outputMapper.Map(subject)
		require.NoError(t, err)
		require.Equal(t, "Foo", result.mappedInputString)
	})
}

type SomeStruct struct {
	SomeField string
	SomeInt   int
}

func TestStructMapper_MapAMapToStruct(t *testing.T) {
	recordInputStringField := gomorph.NewField[any]("InputString")
	recordInputIntField := gomorph.NewField[int]("InputInt")

	outputStringField := gomorph.NewField[string]("SomeField")
	outputIntField := gomorph.NewField[int]("SomeInt")

	// Define the input and output structs
	input := map[string]any{
		"InputString": "hello",
		"InputInt":    42,
	}

	fields := []gomorph.FieldMapper{
		gomorph.From[any, string](recordInputStringField).
			To(outputStringField).
			SkipConversion().
			SkipValidation().
			Build(),
		gomorph.From[int, int](recordInputIntField).
			To(outputIntField).
			SkipConversion().
			SkipValidation().
			Build(),
	}

	output := gomorph.NewStructMapper[map[string]any, SomeStruct](fields)

	result, err := output.Map(input)
	require.NoError(t, err)
	require.Equal(t, "hello", result.SomeField)
	require.Equal(t, 42, result.SomeInt)

}
