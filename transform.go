package gomorph

import (
	"fmt"
)

type KeyLister[K comparable] interface {
	Keys() []K
}

// Generic transformation function type
type TransformFunc[TSource any, TDest any, TMeta any] func(TSource, TMeta) (TDest, error)

type TransformResolver[K comparable, TSource any, TDest any, TMeta any] interface {
	Resolve(key K) (TransformFunc[TSource, TDest, TMeta], bool)
}

type MapResolver[K comparable, TSource any, TDest any, TMeta any] struct {
	mapping map[K]TransformFunc[TSource, TDest, TMeta]
}

func NewMapResolver[K comparable, TSource any, TDest any, TMeta any](
	mapping map[K]TransformFunc[TSource, TDest, TMeta],
) *MapResolver[K, TSource, TDest, TMeta] {
	return &MapResolver[K, TSource, TDest, TMeta]{mapping: mapping}
}

func (r *MapResolver[K, TSource, TDest, TMeta]) Resolve(key K) (TransformFunc[TSource, TDest, TMeta], bool) {
	transform, ok := r.mapping[key]
	return transform, ok
}

func (r *MapResolver[K, TSource, TDest, TMeta]) Keys() []K {
	keys := make([]K, 0, len(r.mapping))
	for k := range r.mapping {
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
	resolver TransformResolver[K, TSource, TDest, TMeta]
	meta     TMeta
	keyFunc  func(TSource) K
}

func NewTransformMapper[K comparable, TSource any, TDest any, TMeta any](
	resolver TransformResolver[K, TSource, TDest, TMeta],
	meta TMeta,
	keyFunc func(TSource) K,
) *TransformMapper[K, TSource, TDest, TMeta] {

	return &TransformMapper[K, TSource, TDest, TMeta]{
		resolver: resolver,
		meta:     meta,
		keyFunc:  keyFunc,
	}
}

func (m *TransformMapper[K, TSource, TDest, TMeta]) SupportedOperations() []K {
	if lister, ok := m.resolver.(KeyLister[K]); ok {
		return lister.Keys()
	}
	return nil
}

func (m *TransformMapper[K, TSource, TDest, TMeta]) Meta() TMeta {
	return m.meta
}

func (m *TransformMapper[K, TSource, TDest, TMeta]) From(source TSource) (TDest, error) {
	key := m.keyFunc(source)
	transform, ok := m.resolver.Resolve(key)
	if !ok {
		var zero TDest
		return zero, fmt.Errorf("no transform for key: %v", key)
	}
	return transform(source, m.meta)
}
