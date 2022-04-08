# Array Utilities
This document describes how to use array utilities in `pkg/shared/utils` folder

## Generic Array

### FindFirst
```go
func FindFirst(predicate func(x interface{}) bool, values ...interface{}) (int, interface{})
```
This function finds the first value `x` into the passed in values where `predicate(x)` is `true`.
It takes a variadic list of `interface{}` so that it can be used with any values

Returned values are:
* the index of the value into the list of passed in values (-1 if not found)
* the value that has been found or `nil` (type: `interface{}`)

Some example usage can be found [here](../pkg/shared/utils/arrays/generic_array_utils_test.go)

## String Array

### FindFirstString
```go
func FindFirstString(values []string, predicate func(x string) bool) int
```
Finds the first `string` into the array matching the given predicate.
If the `string` is found, returns its index, otherwise returns `-1`

### FilterStringSlice
```go
func FilterStringSlice(values []string, predicate func(x string) bool) []string
```
Returns a `slice` containing all the values of `values` that match the predicate `predicate`

### FirstNonEmpty
```go
func FirstNonEmpty(values ...string) (string, error)
```
Returns the first value among the passed in values that is not equal to the empty string ("").
If there are no non-empty strings, it returns an empty string and an error.

This function is useful when we need to assign one value among a list of alternatives.
For example, the following code:
```go
val := option1
if val == "" {
	val = option2
}
if val == "" {
	val = option3
}
if val == "" {
	return fmt.Errorf("Value not found")
}
```
would become:
```go
val, err := arrays.FirstNonEmpty(option1, option2, option3)
if err != nil {
	return err
}
```

### FirstNonEmptyOrDefault
```go
func FirstNonEmptyOrDefault(defaultValue string, values ...string)
```
This function works the same as [FirstNonEmpty](#FirstNonEmpty), but instead of returning an error, returns the default value when no non-empty string can be found
