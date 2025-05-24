package gomorph

import (
	"fmt"
)

// FieldMapper represents an abstract transformation between two fields of potentially different types.
type FieldMapper interface {
	From() Field
	To() Field
	Map(value any) (FieldMappingResult, error)
}

// FieldMapping defines how a value from a source field is transformed and assigned to a target field.
// It links a source field definition, a destination field definition, and a ChainedMapper that performs
// the actual data transformation.
//
// FieldMapping is the core unit of logic for mapping structured fields in struct-to-struct transformations,
// especially when composed using StructMapper or similar high-level constructs.
//
// It supports type-safe configuration through generics and can be composed into broader pipelines.
//
// Example:
//
//	mapping := mapper.NewFieldMapping(
//	    mapper.NewField[string]("source_name"),
//	    mapper.NewField[int]("target_length"),
//	    mapper.NewChainedMapper[string, int]([]mapper.TypedMapper{
//	        StringToIntMapper{},
//	    }),
//	)
//
//	field, value, err := mapping.Map("hello") // value = 5, field = "target_length"
type FieldMapping[TSource, TDest any] struct {
	from  FieldDef[TSource]
	to    FieldDef[TDest]
	using *ChainedMapper[TSource, TDest]
}

func (fm FieldMapping[TSource, TDest]) Using() *ChainedMapper[TSource, TDest] {
	return fm.using
}

func NewFieldMapping[TSource, TDest any](
	from FieldDef[TSource],
	to FieldDef[TDest],
	using *ChainedMapper[TSource, TDest],
) FieldMapping[TSource, TDest] {
	return FieldMapping[TSource, TDest]{
		from:  from,
		to:    to,
		using: using,
	}
}

func (fm FieldMapping[TSource, TDest]) From() Field {
	return fm.from
}

func (fm FieldMapping[TSource, TDest]) To() Field {
	return fm.to
}

func (fm FieldMapping[TSource, TDest]) mapTyped(value TSource) (TDest, error) {
	result, err := fm.Using().Map(value)
	if err != nil {
		return *new(TDest), err
	}
	return result, nil
}

func (fm FieldMapping[TSource, TDest]) Map(value any) (FieldMappingResult, error) {
	castedValue, ok := value.(TSource)
	if !ok {
		err := fmt.Errorf("invalid source type: expected %T, got %T", *new(TSource), value)
		return NewFieldMappingResult(
			fm.To(),
			NewTypedValue(nil),
		), err
	}
	mapped, err := fm.mapTyped(castedValue)
	if err != nil {
		return NewFieldMappingResult(
			fm.To(),
			NewTypedValue(nil),
		), err

	}
	return NewFieldMappingResult(
		fm.To(),
		NewTypedValue(mapped),
	), nil
}
