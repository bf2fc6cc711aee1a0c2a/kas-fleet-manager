package arrays

const ElementNotFound = -1

// PredicateFunc - A function that returns `true` when the received value satisfies some pre-condition
type PredicateFunc[T any] func(x T) bool

// ReducerFunc - function that reduces a given slice of T to a single result of type U
// accumulator - the value of the reduction so far
// currentValue - the current value to be elaborated
type ReducerFunc[T any, U any] func(accumulatorValue U, currentValue T) U

// MapperFunc - A function that receive an object and transform it to something else
// T - the type of the input value
// U - the type of the produced value
type MapperFunc[T any, U any] func(x T) U

// ConsumerFunc - A function that perform some operation on the received object
type ConsumerFunc[T any] func(x T)

// FindFirst Finds the first element into the passed in values where the predicate is true
// Returns the index of the element into the array (or -1 if not found) and the value that has been found
func FindFirst[T any](values []T, predicate PredicateFunc[T]) (int, T) {
	for i, val := range values {
		if predicate(val) {
			return i, val
		}
	}
	var res T
	return ElementNotFound, res
}

// Filter return a slice containing only the values of `values` matching the given predicate
func Filter[T any](values []T, predicate PredicateFunc[T]) []T {
	res := []T{}
	for _, val := range values {
		if predicate(val) {
			val := val
			res = append(res, val)
		}
	}
	return res
}

// Map - creates a new slice where each element is the result of the mapper function to the corresponding element in the input slice
func Map[T any, U any](values []T, mapper MapperFunc[T, U]) []U {
	var res []U
	for _, val := range values {
		res = append(res, mapper(val))
	}
	return res
}

// AnyMatch - check whether at least one of the element of the given slice matches the given predicate
func AnyMatch[T any](values []T, predicate PredicateFunc[T]) bool {
	if idx, _ := FindFirst(values, predicate); idx != -1 {
		return true
	}
	return false
}

// NoneMatch - check that none of the elements of the given slice matches the given predicate
func NoneMatch[T any](values []T, predicate PredicateFunc[T]) bool {
	return !AnyMatch(values, predicate)
}

// AllMatch - check if all the elements of the given slice matches the given predicate
func AllMatch[T any](values []T, predicate PredicateFunc[T]) bool {
	for _, val := range values {
		if !predicate(val) {
			return false
		}
	}
	return true
}

// Reduce - reduces an array to a single value by applying the reducer function.
// initialValue is the starting value of the reduction
// T the type of each element of the input array
// U the type of the result of the reduction
func Reduce[T any, U any](values []T, reducer ReducerFunc[T, U], initialValue U) U {
	res := initialValue
	for _, val := range values {
		val := val
		res = reducer(res, val)
	}
	return res
}

// Contains - check whether the given array contains the given value.
// Each element is compared with the given value using the `==` operator.
func Contains[T comparable](values []T, s T) bool {
	return AnyMatch(values, func(x T) bool { return x == s })
}

// ForEach - runs the given function on each element of the given array
func ForEach[T any](values []T, consumer ConsumerFunc[T]) {
	for i := range values {
		consumer(values[i])
	}
}
