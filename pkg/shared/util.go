package shared

import (
	"io"
	"reflect"
	"strings"
)

// SafeString - Converts a string pointer to a string. If the pointer is `nil`, returns the empty string
func SafeString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

// SafeInt64 - Converts an Int64 pointer to an Int64. If the pointer is `nil`, returns 0
func SafeInt64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// CloseQuietly - Returns a function that closes a `Closer` object without returning any error
func CloseQuietly(c io.Closer) func() {
	return func() {
		_ = c.Close()
	}
}

// IsNotNil - Returns `true` if the passed in value is not nil
func IsNotNil[T any](x T) bool {
	return !IsNil(x)
}

// IsNil - returns `true` if the value is nil
func IsNil[T any](x T) bool {
	v := reflect.ValueOf(x)
	k := v.Kind()
	switch k {
	case reflect.Ptr:
		fallthrough
	case reflect.Interface:
		fallthrough
	case reflect.Slice:
		fallthrough
	case reflect.Map:
		fallthrough
	case reflect.Chan:
		fallthrough
	case reflect.Func:
		return v.IsNil()
	case reflect.Invalid: // naked nil
		return true
	default:
		return false
	}
}

// StringEmpty - returns `true` if the value is nil or is the empty string. Works with both `string` and `*string` objects.
// If `trim` is specified and is `true` trims the string (only first value is considered).
func StringEmpty[T string | *string](x T, trim ...bool) bool {
	trimInput := len(trim) != 0 && trim[0]
	val, ok := any(x).(string)
	if ok {
		if trimInput {
			val = strings.Trim(val, " ")
		}
		return val == ""
	}
	stringPointer, _ := any(x).(*string)
	if trimInput && stringPointer != nil {
		s := strings.Trim(*stringPointer, " ")
		stringPointer = &s
	}
	return stringPointer == nil || *stringPointer == ""
}

// StringEqualsIgnoreCase - Compare 2 strings ignoring the case. Works with both string and *string values.
// If s1 == s2 == nil, returns `true`
func StringEqualsIgnoreCase[T string | *string](s1 T, s2 T) bool {
	if IsNil(s1) && IsNil(s2) {
		return true
	}

	if IsNil(s1) || IsNil(s2) {
		return false
	}

	convertToString := func(v T) string {
		if val, ok := any(v).(string); ok {
			return val
		} else {
			return *any(v).(*string)
		}
	}

	v1 := convertToString(s1)
	v2 := convertToString(s2)

	return strings.EqualFold(v1, v2)
}
