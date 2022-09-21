# Slice Utilities
This document describes how to use array utilities in `pkg/shared/utils` folder.

## Simple API

### FindFirst
```go
func FindFirst[T any](values []T, predicate PredicateFunc[T]) (index int, value T)
```
This function finds the first value `x` into the passed-in values where `predicate(x)` is `true`.

Returned values are:
* The index of the value into the list of passed-in values (-1 if not found)
* The found value or the _zero value_ for the `T` type

Some example usage can be found [here](../pkg/shared/utils/arrays/generic_array_utils_test.go)

### Filter

```go
func Filter[T any](values []T, predicate PredicateFunc[T]) []T
```
This function returns a slice whose values are all the value `x` of the passed in slice (`values`) where `predicate(x)` is `true`

### AnyMatch
```go
func AnyMatch[T any](values []T, predicate PredicateFunc[T]) bool
```
This function checks whether the passed-in slice contains at least one element `x` where `predicate(x)` is `true`.

### NoneMatch
```go
func NoneMatch[T any](values []T, predicate PredicateFunc[T]) bool
```
This function checks whether the passed-in slice does not contain any element `x` where `predicate(x)` is `true`.

### AllMatch
```go
func AllMatch[T any](values []T, predicate PredicateFunc[T]) bool
```
This function checks whether `predicate(x)` is `true` for all the element `x` of the passed-in slice.

### Map
```go
func Map[T any, U any](values []T, mapper MapperFunc[T, U]) []U
```
This function transforms all the elements of a slice of `T`s to a slice of `U`s by applying the `mapper` function to each element.

### Reduce
```go
func Reduce[T any, U any](values []T, reducer ReducerFunc[T, U], initialValue U) U
```
This function reduces the passed-in slice to a single value by applying the reducer function.

### Contains
```go
func Contains[T comparable](values []T, s T) bool
```
This function checks whether the given slice contains the given value.

### ForEach
```go
func ForEach[T any](values []T, consumer ConsumerFunc[T])
```
This function applies the consumer function to each of the elements of the slice.
