package arrays

import "reflect"

// IsNotNilPredicate - returns `true` if the value is not nil
func IsNotNilPredicate[T any](x T) bool {
	return !reflect.ValueOf(x).IsNil()
}

// IsNilPredicate - returns `true` if the value is nil
func IsNilPredicate[T any](x T) bool {
	return reflect.ValueOf(x).IsNil()
}

// StringNotEmptyPredicate - returns `true` if the value is not nil and is not empty string
// Panic if passed value is not a string
func StringNotEmptyPredicate(x any) bool {
	return !StringEmptyPredicate(x)
}

// StringEmptyPredicate - returns `true` if the value is nil or is the empty string
// Panic if passed value is not a string
func StringEmptyPredicate(x any) bool {
	val, ok := x.(string)
	if ok {
		return val == ""
	}
	stringPointer, ok := x.(*string)
	if ok {
		return stringPointer == nil || *stringPointer == ""
	}
	panic("passed value is not a string nor a string pointer")
}

// CompositePredicateAll - returns a composed predicate that will return `true` when all the passed predicates returns `true`
func CompositePredicateAll[T any](predicates ...PredicateFunc[T]) PredicateFunc[T] {
	return func(x T) bool {
		for _, predicate := range predicates {
			if !predicate(x) {
				return false
			}
		}
		return true
	}
}

// CompositePredicateAny - returns a composed predicate that will return `true` when any of the passed predicates returns `true`
func CompositePredicateAny[T any](predicates ...PredicateFunc[T]) PredicateFunc[T] {
	return func(x T) bool {
		for _, predicate := range predicates {
			if predicate(x) {
				return true
			}
		}
		return false
	}
}
