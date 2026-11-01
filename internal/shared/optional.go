package shared

import (
	"encoding/json"
	"fmt"
)

type Optional[T comparable] struct {
	value T
	some  bool
}

func Some[T comparable](v T) Optional[T] {
	return Optional[T]{value: v, some: true}
}

func None[T comparable]() Optional[T] {
	return Optional[T]{}
}

func SomeIf[T comparable](v T, ok bool) Optional[T] {
	if ok {
		return Some(v)
	}
	return None[T]()
}

func FromPtr[T comparable](p *T) Optional[T] {
	if p == nil {
		return None[T]()
	}
	return Some(*p)
}

func (o Optional[T]) Equal(other Optional[T]) bool {
	return o.value == other.value
}

func FromMap[K comparable, V comparable](m map[K]V, key K) Optional[V] {
	v, ok := m[key]
	return SomeIf(v, ok)
}

func (o Optional[T]) TryUnwrap() (T, bool) {
	return o.value, o.some
}

func (o Optional[T]) Unwrap() T {
	if !o.some {
		panic("optional: Unwrap() called on nil value (None)")
	}
	return o.value
}

func (o Optional[T]) UnwrapOr(fallback T) T {
	if o.some {
		return o.value
	}
	return fallback
}

func (o Optional[T]) UnwrapOrElse(fn func() T) T {
	if o.some {
		return o.value
	}
	return fn()
}

func (o Optional[T]) IsSome() bool { return o.some }

func (o Optional[T]) IsNone() bool { return !o.some }

func (o Optional[T]) IsSomeAnd(pred func(T) bool) bool {
	return o.some && pred(o.value)
}

func (o Optional[T]) Filter(pred func(T) bool) Optional[T] {
	if o.some && pred(o.value) {
		return o
	}
	return None[T]()
}

func (o Optional[T]) Or(other Optional[T]) Optional[T] {
	if o.some {
		return o
	}
	return other
}

func (o Optional[T]) OrElse(fn func() Optional[T]) Optional[T] {
	if o.some {
		return o
	}
	return fn()
}

func (o Optional[T]) ToPtr() *T {
	if o.some {
		return &o.value
	}
	return nil
}

func Map[T, U comparable](o Optional[T], fn func(T) U) Optional[U] {
	if o.some {
		return Some(fn(o.value))
	}
	return None[U]()
}

func FlatMap[T, U comparable](o Optional[T], fn func(T) Optional[U]) Optional[U] {
	if o.some {
		return fn(o.value)
	}
	return None[U]()
}

func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.some {
		return []byte("null"), nil
	}
	return json.Marshal(o.value)
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*o = None[T]()
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	*o = Some(v)
	return nil
}

func (o Optional[T]) String() string {
	if o.some {
		return fmt.Sprintf("Some(%v)", o.value)
	}
	return "None"
}
