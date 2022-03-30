package arrays

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestGenericFindFirstWithStrings(t *testing.T) {
	RegisterTestingT(t)
	idx, val := FindFirst(func(x interface{}) bool { return x.(string) == "Red" }, "This", "Is", "Red", "Hat")
	Expect(idx).To(Equal(2))
	Expect(val).To(Equal("Red"))
}

func TestGenericFindFirstWithInt(t *testing.T) {
	RegisterTestingT(t)
	idx, val := FindFirst(func(x interface{}) bool { return x.(int) > 5 }, 1, 2, 3, 4, 5, 6, 7, 8)
	Expect(idx).To(Equal(5))
	Expect(val).To(Equal(6))
}

func TestGenericWithIncompatibleTypes(t *testing.T) {
	RegisterTestingT(t)
	idx, val := FindFirst(func(x interface{}) bool {
		if val, ok := x.(int); ok {
			return val > 5
		} else {
			return false
		}
	}, 1, 2, "Hello", 4, 5, 6, 7, 8)
	Expect(idx).To(Equal(5))
	Expect(val).To(Equal(6))
}
