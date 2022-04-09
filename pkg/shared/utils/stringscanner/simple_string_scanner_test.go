package stringscanner

import (
	. "github.com/onsi/gomega"
	"testing"
)

func Test_SimpleScanner(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		expectedTokens []Token
	}{
		{
			name:           "Testing empty string",
			value:          "",
			expectedTokens: []Token{},
		},
		{
			name:  "Testing 1 token",
			value: "a",
			expectedTokens: []Token{
				{TokenType: ALPHA, Value: "a", Position: 0},
			},
		},
		{
			name:  "Testing 5 tokens",
			value: "ab(1.",
			expectedTokens: []Token{
				{TokenType: ALPHA, Value: "a", Position: 0},
				{TokenType: ALPHA, Value: "b", Position: 1},
				{TokenType: SYMBOL, Value: "(", Position: 2},
				{TokenType: DIGIT, Value: "1", Position: 3},
				{TokenType: DECIMALPOINT, Value: ".", Position: 4},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			scanner := NewSimpleScanner()
			scanner.Init(tt.value)
			allTokens := []Token{}
			for scanner.Next() {
				allTokens = append(allTokens, *scanner.Token())
			}
			Expect(allTokens).To(Equal(tt.expectedTokens))
		})
	}
}
