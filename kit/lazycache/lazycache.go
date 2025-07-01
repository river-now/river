// Useful for creating methods that lazily initialize a derived value
// and run only once no matter how many times the method is called.
// Simply add a private field to your struct of type Value[T], and
// then return the value from a public getter method using Get[T].
package lazycache

import "sync"

type Value[T any] struct {
	val  T
	init sync.Once
}

func Get[T any](v *Value[T], initFunc func() T) T {
	v.init.Do(func() { v.val = initFunc() })
	return v.val
}
