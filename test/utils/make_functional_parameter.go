package utils

import "reflect"

type FunctionalParameter[T any] func(request *T)

// MakeFunctionalParameter creates a functional parameter for the specified struct field and with the specified value type
// T: is the type of the functional parameter
// dest: is the name of the field that the functional parameter will set
func MakeFunctionalParameter[T any, U any](dest string) func(value T) FunctionalParameter[U] {
	return func(value T) FunctionalParameter[U] {
		return func(request *U) {
			r := reflect.ValueOf(request)
			reflect.Indirect(r).FieldByName(dest).Set(reflect.ValueOf(value))
		}
	}
}
