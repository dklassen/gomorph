package gomorph

type IdentityMapper[T any] struct {
	TypeMap[T, T]
}

func (m IdentityMapper[T]) From(source any) (any, error) {
	return source.(T), nil
}
