package gomorph

import (
	"fmt"
	"reflect"
)

type Record = map[string]any

type Mapper[TSource, TDest any] interface {
	From(source TSource) (TDest, error)
}

type BidirectionalMapper[TSource, TDest any] interface {
	Mapper[TSource, TDest]
	To(dest TDest) (TSource, error)
}

type TypeInfo interface {
	SourceType() reflect.Type
	TargetType() reflect.Type
}

// TypedMapper represents a runtime-composable step that has access to a value
// from one type to another. It combines a dynamic From() function with runtime type metadata,
// enabling type-safe chaining of arbitrary transformations in mapper pipelines.
//
// Each TypedMapper implementation must specify its input and output types via the TypeInfo interface,
// allowing validation of compatibility when building chained mappers.
//
// This interface is typically implemented by embedding TypeMap[TSource, TDest] and defining a
// From(source any) (any, error) method.
//
// Example:
//
//	type StringToIntMapper struct {
//	    mapper.TypeMap[string, int]
//	}
//
//	func (m StringToIntMapper) From(source any) (any, error) {
//	    s, ok := source.(string)
//	    if !ok {
//	        return 0, fmt.Errorf("expected string, got %T", source)
//	    }
//	    return len(s), nil
//	}
type TypedMapper interface {
	Mapper[any, any]
	TypeInfo
}

// TypeMap is a zero-value helper type used to represent the source and target types
// of a TypedMapper at runtime. It provides the type information needed for dynamic
// composition of mappers via reflection, without implementing any actual transformation logic.
//
// This type is typically embedded in concrete mappers to fulfill the TypeInfo interface,
// allowing mapper chains to validate that input and output types line up across steps.
//
// Example usage:
//
//	type MyMapper struct {
//	    mapper.TypeMap[string, int] // declares MyMapper as transforming string -> int
//	}
//
//	func (m MyMapper) From(source any) (any, error) {
//	    // implementation...
//	}
type TypeMap[TSource, TDest any] struct{}

func (t TypeMap[TSource, TDest]) SourceType() reflect.Type {
	var zero TSource
	return reflect.TypeOf(zero)
}

func (t TypeMap[TSource, TDest]) TargetType() reflect.Type {
	var zero TDest
	return reflect.TypeOf(zero)
}

// Slice[T] is a type constraint that restricts T to be a slice of any type.
type Slice[T any] interface {
	~[]T
}

// EnsureSlice is a helper function that panics if the type of T is not a slice during runtime.
func EnsureSlice[T any]() {
	var zero T
	if reflect.TypeOf(zero).Kind() != reflect.Slice {
		panic(fmt.Sprintf("EnsureSlice: expected slice type, got %T", zero))
	}
}

// SliceMapper is a TypedMapper that transforms slices of one type to slices of another.
// it uses an element mapper to transform each element in the source slice to the target type.
// the element mapper must implement the TypedMapper interface.
type SliceMapper[TSource Slice[T], TDest Slice[D], T, D any] struct {
	elementMapper TypedMapper
}

// This craziness lets us restrict to a slice of whatever type but its verbose and annoying
// func NewSliceMapper[TSource Slice[TElement], TElement any, TDest Slice[TDelement], TDelement any](elementMapper TypedMapper) *SliceTypedMapper[TSource, TDest] {
func NewSliceMapper[TSource Slice[T], TDest Slice[D], T, D any](elementMapper TypedMapper) *SliceMapper[TSource, TDest, T, D] {
	EnsureSlice[TSource]()
	EnsureSlice[TDest]()

	return &SliceMapper[TSource, TDest, T, D]{
		elementMapper: elementMapper,
	}
}

func (stm *SliceMapper[Tsource, TDest, T, D]) SourceType() reflect.Type {
	var zero Tsource
	return reflect.TypeOf(zero)
}

func (stm *SliceMapper[Tsource, TDest, T, D]) TargetType() reflect.Type {
	var zero T
	return reflect.TypeOf(zero)
}

func (stm *SliceMapper[TSource, TDest, T, D]) From(source any) (any, error) {
	castedSource, ok := source.([]T)
	if !ok {
		return nil, fmt.Errorf("invalid source type: expected %T, got %T", *new(TSource), source)
	}

	var result TDest
	for _, element := range castedSource {
		transformed, err := stm.elementMapper.From(element)
		if err != nil {
			return nil, err
		}
		result = append(result, transformed.(D))
	}

	return result, nil
}

// ChainedMapper composes multiple TypedMapper instances into a single transformation pipeline,
// where the output of one mapper is passed as the input to the next.
//
// It is generic over the initial source type and final destination type, but internally supports
// dynamically typed steps via reflection. Type compatibility between each adjacent mapper is
// validated at runtime based on their declared SourceType and TargetType.
//
// The chain is evaluated left to right: mapper₁ ∘ mapper₂ ∘ ... ∘ mapperₙ, where mapper₁ receives
// the original input and mapperₙ produces the final output.
//
// Example:
//
//	mappers := []mapper.TypedMapper{
//	    StringToIntMapper{},  // string -> int
//	    IntDoubler{},         // int -> int
//	}
//
//	chained := mapper.NewChainedMapper[string, int](mappers)
//	result, err := chained.From("hello") // result is 10 if len("hello") == 5
type ChainedMapper[TSource, TDest any] struct {
	mappers []TypedMapper
}

// NewChainedMapper creates a new composition chain of mappers.
func NewChainedMapper[TSource, TDest any](mappers ...TypedMapper) *ChainedMapper[TSource, TDest] {
	if len(mappers) == 0 {
		return &ChainedMapper[TSource, TDest]{mappers: mappers}
	}

	var sourceType TSource
	expectedSourceType := reflect.TypeOf(sourceType)

	if mappers[0].SourceType() != expectedSourceType {
		panic(fmt.Sprintf("first mapper must accept %T, got %v", sourceType, mappers[0].SourceType()))
	}

	var destType TDest
	expectedDestType := reflect.TypeOf(destType)
	if mappers[len(mappers)-1].TargetType() != expectedDestType {
		panic(fmt.Sprintf("last mapper must produce %T, got %v", destType, mappers[len(mappers)-1].TargetType()))
	}

	lenMappers := len(mappers)
	for i, m := range mappers {
		if i+1 >= lenMappers {
			continue
		}

		if m.TargetType() != mappers[i+1].SourceType() {
			panic(fmt.Sprintf("type mismatch between mapper %d output and mapper %d input", i, i+1))
		}
	}

	return &ChainedMapper[TSource, TDest]{mappers: mappers}
}

func (c *ChainedMapper[TSource, TDest]) Map(input TSource) (TDest, error) {
	var err error
	var current any = input
	for i, m := range c.mappers {
		current, err = m.From(current)
		if err != nil {
			var zero TDest
			return zero, fmt.Errorf("mapper chain failed at step %d: %w", i+1, err)
		}
	}

	result, ok := current.(TDest)
	if !ok {
		var zero TDest
		return zero, fmt.Errorf("final type mismatch: expected %T, got %T", zero, current)
	}
	return result, nil
}

// StructMapper represents a composite field-level mapper for complex structured types.
// It manages a set of individual FieldMapper instances, each responsible for transforming a
// specific field from the source type to the destination type.
//
// This struct is intended to be embedded in higher-level mappers and used within ChainedMapper
// pipelines as a reusable mapping unit. It supports nesting of complex mappings and enables
// recursive transformation of structured data.
//
// Consumers must provide a concrete From implementation that coordinates mapping logic
// across fields, typically by invoking each FieldMapper in the fieldMappings map.
//
// Example:
//
//	type UserMapper struct {
//	    mapper.StructMapper[UserDTO, UserModel]
//	}
//
//	func (m UserMapper) From(dto UserDTO) (UserModel, error) {
//	    // collect field mappings, apply them to dto, build the result...
//	}
//
//	// Used as a step in a chain:
//	chained := mapper.NewChainedMapper[UserDTO, SomeOutputType]([]mapper.TypedMapper{
//	    UserMapper{},
//	    SomeOtherMapper{},
//	})
type StructMapper[TSource, TDest any] struct {
	fieldMappings []FieldMapper
}

// TODO:: Allow type Keys to be used as keys
// THis way we can lookup by required struct type to avoid having to know the name of thing we
// are looking up
//func (b *StructMapper[TSource, TDest]) Using(dest string) FieldMapper {
//	return b.fieldMappings[dest]
//}

// func (b *StructMapper[TSource, TDest]) From(source TSource) (TDest, error) {
// 	var zero TDest
// 	return zero, fmt.Errorf("From must be implemented by concrete mapper")
// }

func (b *StructMapper[TSource, TDest]) From(input TSource) (TDest, error) {
	var output TDest
	err := mapStruct(input, &output, b.fieldMappings)
	if err != nil {
		return output, err
	}
	return output, nil
}

func NewStructMapper[TSource, TDest any](mappings []FieldMapper) StructMapper[TSource, TDest] {

	return StructMapper[TSource, TDest]{
		fieldMappings: mappings,
	}
}

func assignValue(obj any, to string, value any) error {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	field := val.FieldByName(to)
	if field.IsValid() && field.CanSet() {
		v := reflect.ValueOf(value)
		if !v.Type().AssignableTo(field.Type()) {
			return fmt.Errorf("type mismatch: cannot assign %v to %v", v.Type(), field.Type())
		}
		field.Set(v)
		return nil
	}

	method := reflect.ValueOf(obj).MethodByName(to)
	if method.IsValid() && method.Type().NumIn() == 1 {
		argType := method.Type().In(0)
		v := reflect.ValueOf(value)
		if !v.Type().AssignableTo(argType) {
			return fmt.Errorf("cannot assign value of type %v to method %q expecting %v", v.Type(), to, argType)
		}
		method.Call([]reflect.Value{v})
		return nil
	}

	return fmt.Errorf("could not assign or call method for %s", to)
}

func getFieldValueByName(obj any, name string) (any, error) {
	val := reflect.ValueOf(obj)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() == reflect.Struct {
		if field := val.FieldByName(name); field.IsValid() && field.CanInterface() {
			return field.Interface(), nil
		}
	}

	if val.Kind() == reflect.Map {
		if field := val.MapIndex(reflect.ValueOf(name)); field.IsValid() {
			fmt.Println("found field", field)
			return field.Interface(), nil
		}
	}

	ptr := reflect.ValueOf(obj)
	if ptr.Kind() != reflect.Ptr {
		copy := reflect.New(ptr.Type()).Elem()
		copy.Set(ptr)
		ptr = copy.Addr()
	}

	method := ptr.MethodByName(name)
	if method.IsValid() && method.Type().NumIn() == 0 && method.Type().NumOut() == 1 {
		return method.Call(nil)[0].Interface(), nil
	}

	return nil, fmt.Errorf("field or zero-arg getter %q not found on %T", name, obj)
}

func mapStruct[I any, O any](input I, output O, mappings []FieldMapper) error {
	for _, fieldMapper := range mappings {
		fromName := fieldMapper.From().Name()
		toName := fieldMapper.To().Name()

		rawValue, err := getFieldValueByName(input, fromName)
		if err != nil {
			return fmt.Errorf("input error [%s]: %w", fromName, err)
		}

		mapped, err := fieldMapper.Map(rawValue)
		if err != nil {
			return fmt.Errorf("mapping error [%s]: %w", fromName, err)
		}

		err = assignValue(output, toName, mapped.MappedValue().Value())
		if err != nil {
			return fmt.Errorf("output error [%s]: %w", toName, err)
		}
	}
	return nil
}

func GetField[T any](record map[string]any, field FieldDef[T]) (T, error) {
	val, ok := record[field.Name()]
	if !ok {
		var zero T
		return zero, &ValidationError{
			Field:   field.Name(),
			Value:   val,
			Message: fmt.Sprintf("field %q not found", field.Name()),
		}
	}

	typedVal, ok := val.(T)
	if !ok {
		var zero T
		return zero, &ValidationError{
			Field:   field.Name(),
			Value:   val,
			Message: fmt.Sprintf("expected type %T, got %T", zero, val),
		}
	}

	return typedVal, nil
}
