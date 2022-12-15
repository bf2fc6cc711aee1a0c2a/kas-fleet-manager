package arrays

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
)

// IsNotNilPredicate - returns `true` if the value is not nil
func IsNotNilPredicate[T any](x T) bool {
	return !shared.IsNil(x)
}

// IsNilPredicate - returns `true` if the value is nil
func IsNilPredicate[T any](x T) bool {
	return shared.IsNil(x)
}

// StringNotEmptyPredicate - returns `true` if the value is not nil and is not empty string
func StringNotEmptyPredicate[T string | *string](x T) bool {
	return !shared.StringEmpty(x)
}

// StringEmptyPredicate - returns `true` if the value is nil or is the empty string
func StringEmptyPredicate[T string | *string](x T) bool {
	return shared.StringEmpty(x)
}

func EqualsPredicate[T comparable](value T) PredicateFunc[T] {
	return func(x T) bool {
		return x == value
	}
}

func StringEqualsIgnoreCasePredicate[T string | *string](value T) PredicateFunc[T] {
	return func(x T) bool {
		return shared.StringEqualsIgnoreCase(x, value)
	}
}

func StringHasPrefixIgnoreCasePredicate[T string | *string](value T) PredicateFunc[T] {
	return func(prefix T) bool {
		return shared.StringHasPrefixIgnoreCase(value, prefix)
	}
}

func StringHasNotPrefixIgnoreCasePredicate[T string | *string](value T) PredicateFunc[T] {
	return func(prefix T) bool {
		return !shared.StringHasPrefixIgnoreCase(value, prefix)
	}
}

func StringHasSuffixIgnoreCasePredicate[T string | *string](value T) PredicateFunc[T] {
	return func(suffix T) bool {
		return shared.StringHasSuffixIgnoreCase(value, suffix)
	}
}

func StringHasNotSuffixIgnoreCasePredicate[T string | *string](value T) PredicateFunc[T] {
	return func(suffix T) bool {
		return !shared.StringHasSuffixIgnoreCase(value, suffix)
	}
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
