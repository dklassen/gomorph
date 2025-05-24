package gomorph_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/dklassen/gomorph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var sourceField = gomorph.NewField[string]("source_field")
var targetStringField = gomorph.NewField[string]("target_string_field")

type MockStringToStringMapper struct {
	mock.Mock
	gomorph.TypeMap[string, string]
}

func (m *MockStringToStringMapper) From(source any) (any, error) {
	args := m.Called(source)
	return args.Get(0), args.Error(1)
}

type PointerMockStringToStringMapper struct {
	mock.Mock
	gomorph.TypeMap[*string, *string]
}

func (m *PointerMockStringToStringMapper) From(source any) (any, error) {
	args := m.Called(source)
	return args.Get(0), args.Error(1)
}

func TestFieldMapping_SuccessfulMap(t *testing.T) {
	mockMapper := new(MockStringToStringMapper)
	mapping := gomorph.NewFieldMapping(
		sourceField,
		targetStringField,
		gomorph.NewChainedMapper[string, string](
			mockMapper,
		),
	)

	mockMapper.On("From", "source_value").Return("mapped_value", nil)

	result, _ := mapping.Map("source_value")

	expected := gomorph.NewTypedValue("mapped_value")

	assert.Equal(t, targetStringField, result.TargetField())
	assert.Equal(t, expected, result.MappedValue())
	mockMapper.AssertExpectations(t)
}

func TestFieldMapping_Map_InvalidType(t *testing.T) {
	mockMapper := new(MockStringToStringMapper)
	mapping := gomorph.NewFieldMapping(
		sourceField,
		targetStringField,
		gomorph.NewChainedMapper[string, string](
			mockMapper,
		),
	)

	_, err := mapping.Map(123)

	assert.Error(t, err)
}

func TestFieldMapping_Map_MapperError(t *testing.T) {
	mockMapper := new(MockStringToStringMapper)
	mapping := gomorph.NewFieldMapping(
		sourceField,
		targetStringField,
		gomorph.NewChainedMapper[string, string](
			mockMapper,
		),
	)

	expectedErr := errors.New("gomorph error")
	mockMapper.On("From", "source_value").Return("empty", expectedErr)

	_, err := mapping.Map("source_value")

	assert.Error(t, err)
	mockMapper.AssertExpectations(t)
}

func TestFieldMapping_Map_EmptySourceValue(t *testing.T) {
	mockMapper := new(MockStringToStringMapper)
	mapping := gomorph.NewFieldMapping(
		sourceField,
		targetStringField,
		gomorph.NewChainedMapper[string, string](
			mockMapper,
		),
	)

	mockMapper.On("From", "").Return("", nil)

	result, err := mapping.Map("")
	expected := gomorph.NewTypedValue("")

	assert.NoError(t, err)
	assert.Equal(t, targetStringField, result.TargetField())
	assert.Equal(t, expected, result.MappedValue())
	mockMapper.AssertExpectations(t)
}

func TestFieldMapping_Map_NilInput(t *testing.T) {
	mockMapper := new(PointerMockStringToStringMapper)
	srcField := gomorph.NewField[*string]("source_ptr_field")
	tgtField := gomorph.NewField[*string]("target_ptr_field")

	mapping := gomorph.NewFieldMapping(
		srcField,
		tgtField,
		gomorph.NewChainedMapper[*string, *string](
			mockMapper,
		),
	)

	var nilPtr *string
	mockMapper.On("From", nilPtr).Return(nilPtr, nil)

	result, err := mapping.Map(nilPtr)
	expected := gomorph.NewTypedValue(nilPtr)

	assert.NoError(t, err)
	assert.Equal(t, expected, result.MappedValue())
	assert.Equal(t, tgtField, result.TargetField())
	mockMapper.AssertExpectations(t)
}

type TrimMapper struct {
	gomorph.TypeMap[string, string]
}

func (m TrimMapper) From(source any) (any, error) {
	return strings.TrimSpace(source.(string)), nil
}

type UppercaseMapper struct {
	gomorph.TypeMap[string, string]
}

func (m UppercaseMapper) From(source any) (any, error) {
	return strings.ToUpper(source.(string)), nil
}

type CountryCodeMapper struct {
	normalizations map[string]string
	gomorph.TypeMap[string, string]
}

func NewCountryCodeMapper() *CountryCodeMapper {
	return &CountryCodeMapper{
		normalizations: map[string]string{
			"US": "USA",
			"UK": "GBR",
			"GB": "GBR",
		},
	}
}

func (m CountryCodeMapper) From(source any) (any, error) {
	if normalized, ok := m.normalizations[source.(string)]; ok {
		return normalized, nil
	}
	return source, nil
}

func TestFieldMapping_MultipleMappers_RealTransformations(t *testing.T) {
	srcField := gomorph.NewField[string]("country_code")
	targetField := gomorph.NewField[string]("normalized_country_code")

	mapping := gomorph.NewFieldMapping(
		srcField,
		targetField,
		gomorph.NewChainedMapper[string, string](
			&TrimMapper{},
			&UppercaseMapper{},
			NewCountryCodeMapper(),
		),
	)

	tests := []struct {
		name     string
		input    string
		expected gomorph.TypedValue
	}{
		{
			name:     "trim and normalize US code",
			input:    "  us  ",
			expected: gomorph.NewTypedValue("USA"),
		},
		{
			name:     "normalize UK variations",
			input:    "UK",
			expected: gomorph.NewTypedValue("GBR"),
		},
		{
			name:     "handle GB variation",
			input:    "GB",
			expected: gomorph.NewTypedValue("GBR"),
		},
		{
			name:     "pass through unknown code",
			input:    "FRA",
			expected: gomorph.NewTypedValue("FRA"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := mapping.Map(tt.input)

			assert.NoError(t, err)
			assert.Equal(t, targetField, result.TargetField())
			assert.Equal(t, tt.expected, result.MappedValue())
		})
	}
}
