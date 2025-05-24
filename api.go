package gomorph

// NOTE:: We can change but this is to help with making sure people do the right thing conciously
// and not accidentally mix up the steps.
type FromStep[TSource, TDest any] interface {
	To(field FieldDef[TDest]) ConvertStep[TSource, TDest]
}

type ConvertStep[TSource, TDest any] interface {
	ConvertWith(TypeConverter) ValidateStep[TSource, TDest]
	SkipConversion() ValidateStep[TSource, TDest]
}

type ValidateStep[TSource, TDest any] interface {
	ValidateWith(Validator) BuildStep[TSource, TDest]
	SkipValidation() BuildStep[TSource, TDest]
}

type BuildStep[TSource, TDest any] interface {
	Build() FieldMapping[TSource, TDest]
}

type TypeConverter interface {
	TypedMapper // same same, but different semantic "constraints"
}

type Validator interface {
	TypedMapper // same same, but with different semantic "constraints"
}

// FieldMappingBuilder provides a fluent API to construct a FieldMapping.
// It allows specifying a source field, destination field, optional type converters, and validators.
type FieldMappingBuilder[TSource, TDest any] struct {
	from       FieldDef[TSource]
	to         FieldDef[TDest]
	validate   Validator
	modifyType TypeConverter
}

// From begins the construction of a FieldMappingBuilder with a source field.
// It sets the source type and initializes the destination type to the same type.
//
// Example:
//
//	builder := gomorph.From(gomorph.NewField[string]("source_name"))
func From[TSource, TDest any](field FieldDef[TSource]) FromStep[TSource, TDest] {
	return &FieldMappingBuilder[TSource, TDest]{from: field}
}

// To sets the destination field for the FieldMappingBuilder.
// This determines the target field and type for the mapping.
//
// Example:
//
//	builder := gomorph.From(sourceField).To(targetField)
func (b *FieldMappingBuilder[TSource, TDest]) To(field FieldDef[TDest]) ConvertStep[TSource, TDest] {
	b.to = field
	return b
}

// ValidateWith attaches a Validator to the FieldMappingBuilder.
// This function will be called on the value after it has been transformed.
// It is optional; omit it if no validation is needed.
//
// Example:
//
//	builder := builder.ValidateWith(NonNegativeValidator{})
func (b *FieldMappingBuilder[TSource, TDest]) ValidateWith(validator Validator) BuildStep[TSource, TDest] {
	b.validate = validator
	return b
}

// ConvertWith attaches a TypeConverter to the FieldMappingBuilder.
// This function transforms the input value before validation is performed.
//
// Example:
//
//	builder := builder.ConvertWith(StringToIntConverter{})
func (b *FieldMappingBuilder[TSource, TDest]) ConvertWith(modifier TypeConverter) ValidateStep[TSource, TDest] {
	b.modifyType = modifier
	return b
}

func (b *FieldMappingBuilder[TSource, TDest]) SkipConversion() ValidateStep[TSource, TDest] {
	return b
}

func (b *FieldMappingBuilder[TSource, TDest]) SkipValidation() BuildStep[TSource, TDest] {
	return b
}

// Build finalizes the builder into a FieldMapping.
// It constructs the underlying ChainedMapper using any attached converter and validator.
// The resulting FieldMapping can then be used to transform and assign field values.
//
// Example:
//
//	mapping := builder.Build()
func (b *FieldMappingBuilder[TSource, TDest]) Build() FieldMapping[TSource, TDest] {
	var mappers []TypedMapper
	if b.modifyType != nil {
		mappers = append(mappers, b.modifyType)
	}
	if b.validate != nil {
		mappers = append(mappers, b.validate)
	}

	return NewFieldMapping(
		b.from,
		b.to,
		NewChainedMapper[TSource, TDest](mappers...),
	)
}
