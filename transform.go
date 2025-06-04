package gomorph

import (
	"fmt"
)

// Generic transformation function type
type TransformFunc[TSource any, TDest any, TMeta any] func(TSource, TMeta) (TDest, error)

type TransformEntry[TSource any, TDest any, TMeta any] struct {
	Transform TransformFunc[TSource, TDest, TMeta]
	Meta      TMeta
}

// Generic transformation map type
type TransformMap[K comparable, TSource any, TDest any, TMeta any] map[K]TransformEntry[TSource, TDest, TMeta]

//	SupportedOperations returns a slice of keys representing the operations that are registered in the TransformMap.
//
// Returns an unordered list of keys .
func (m TransformMap[K, TSource, TDest, TMeta]) SupportedOperations() []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

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
type TransformMapper[K comparable, TSource any, TDest any, TMeta any] struct {
	transforms TransformMap[K, TSource, TDest, TMeta]
	keyFunc    func(TSource) K
}

func NewTransformMapper[K comparable, TSource any, TDest any, TMeta any](
	transforms TransformMap[K, TSource, TDest, TMeta],
	keyFunc func(TSource) K,
) *TransformMapper[K, TSource, TDest, TMeta] {
	return &TransformMapper[K, TSource, TDest, TMeta]{transforms: transforms, keyFunc: keyFunc}
}

func (m *TransformMapper[K, TSource, TDest, TMeta]) From(source TSource) (TDest, error) {
	key := m.keyFunc(source)
	entry, ok := m.transforms[key]
	if !ok {
		var zero TDest
		return zero, fmt.Errorf("no transform for key: %v", key)
	}
	return entry.Transform(source, entry.Meta)
}
