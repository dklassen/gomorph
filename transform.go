package gomorph

import (
	"fmt"
)

// Generic transformation function type
type TransformFunc[TSource any, TDest any] func(TSource) TDest

// Generic transformation map type
type TransformMap[K comparable, TSource any, TDest any] map[K]TransformFunc[TSource, TDest]

// Generic mapper using a key to select the transformation and perform dispatch
// on it. This is really useful for cases where you have a full struct transformation.

// This is useful for scenarios where the transformation logic depends on a property 
// of the source value (such as an operator, type, or tag), allowing you to easily
// dispatch to the correct transformation function based on that property.
//
// Example usage:
//
//	transforms := TransformMapper[string, Input, Output]{
//	    "foo": func(i Input) Output { ... },
//	    "bar": func(i Input) Output { ... },
//	}
//	mapper := NewTransformMapper(transforms, func(i Input) string { return i.Kind })
//	out, err := mapper.From(input)
type TransformMapper[K comparable, TSource any, TDest any] struct {
	transforms TransformMap[K, TSource, TDest]
	keyFunc    func(TSource) K
}

func NewTransformMapper[K comparable, TSource any, TDest any](
	transforms TransformMap[K, TSource, TDest],
	keyFunc func(TSource) K,
) *TransformMapper[K, TSource, TDest] {
	return &TransformMapper[K, TSource, TDest]{transforms: transforms, keyFunc: keyFunc}
}

func (m *TransformMapper[K, TSource, TDest]) From(source any) (any, error) {
	s, ok := source.(TSource)
	if !ok {
		return nil, fmt.Errorf("expected %T, got %T", *new(TSource), source)
	}
	key := m.keyFunc(s)
	transform, ok := m.transforms[key]
	if !ok {
		return nil, fmt.Errorf("no transform for key: %v", key)
	}
	return transform(s), nil
}
