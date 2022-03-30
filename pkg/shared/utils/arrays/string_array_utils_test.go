package arrays

import (
	. "github.com/onsi/gomega"
	"testing"
	"unicode"
)

func TestStringFindFirst(t *testing.T) {
	RegisterTestingT(t)
	slice := []string{"This", "Is", "Red", "Hat"}
	idx := FindFirstString(slice, func(x string) bool { return x == "Red" })
	Expect(idx).To(Equal(2))
	Expect(slice[idx]).To(Equal("Red"))
}

func TestFilterStringSlice(t *testing.T) {
	RegisterTestingT(t)
	res := FilterStringSlice([]string{"this", "is", "Red", "Hat"}, func(x string) bool { return x != "" && unicode.IsUpper([]rune(x)[0]) })
	Expect(res).To(HaveLen(2))
	Expect(res).To(Equal([]string{"Red", "Hat"}))
}

func TestStringFindFirst_NotFound(t *testing.T) {
	RegisterTestingT(t)
	slice := []string{"This", "Is", "Red", "Hat"}
	idx := FindFirstString(slice, func(x string) bool { return x == "Blue" })
	Expect(idx).To(Equal(-1))
}

func TestStringFirstNonEmpty(t *testing.T) {
	RegisterTestingT(t)
	val, err := FirstNonEmpty("", "", "", "Red", "Hat")
	Expect(val).To(Equal("Red"))
	Expect(err).ToNot(HaveOccurred())
}

func TestStringFirstNonEmpty_NotFound(t *testing.T) {
	RegisterTestingT(t)
	val, err := FirstNonEmpty("", "", "", "", "")
	Expect(val).To(Equal(""))
	Expect(err).To(HaveOccurred())
}

func TestStringFirstNonEmpty_NoValues(t *testing.T) {
	RegisterTestingT(t)
	val, err := FirstNonEmpty()
	Expect(val).To(Equal(""))
	Expect(err).To(HaveOccurred())
}

func TestStringFirstNonEmptyOrDefault(t *testing.T) {
	RegisterTestingT(t)
	val := FirstNonEmptyOrDefault("Red Hat", "", "", "Red", "Hat")
	Expect(val).To(Equal("Red"))
}

func TestStringFirstNonEmptyOrDefault_Notfound(t *testing.T) {
	RegisterTestingT(t)
	val := FirstNonEmptyOrDefault("Red Hat", "", "", "", "")
	Expect(val).To(Equal("Red Hat"))
}
