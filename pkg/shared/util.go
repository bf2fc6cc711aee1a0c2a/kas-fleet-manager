package shared

import "io"

func SafeString(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func SafeInt64(ptr *int64) int64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

func CloseQuietly(c io.Closer) func() {
	return func() {
		_ = c.Close()
	}
}
