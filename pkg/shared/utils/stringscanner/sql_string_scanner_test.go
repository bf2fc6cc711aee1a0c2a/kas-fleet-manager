package stringscanner

import (
	. "github.com/onsi/gomega"
	"testing"
)

func Test_SQLStringScanner(t *testing.T) {
	tests := []struct {
		name           string
		value          string
		expectedTokens []Token
	}{{
		name:  "Test simple select",
		value: "SELECT * FROM ADDRESS_BOOK",
		expectedTokens: []Token{
			{TokenType: LITERAL, Value: "SELECT", Position: 0},
			{TokenType: LITERAL, Value: "*", Position: 7},
			{TokenType: LITERAL, Value: "FROM", Position: 9},
			{TokenType: LITERAL, Value: "ADDRESS_BOOK", Position: 14},
		},
	},
		{
			name:  "Test quoted string",
			value: "SELECT * FROM ADDRESS_BOOK WHERE SURNAME = 'surname with spaces'",
			expectedTokens: []Token{
				{TokenType: LITERAL, Value: "SELECT", Position: 0},
				{TokenType: LITERAL, Value: "*", Position: 7},
				{TokenType: LITERAL, Value: "FROM", Position: 9},
				{TokenType: LITERAL, Value: "ADDRESS_BOOK", Position: 14},
				{TokenType: LITERAL, Value: "WHERE", Position: 27},
				{TokenType: LITERAL, Value: "SURNAME", Position: 33},
				{TokenType: OP, Value: "=", Position: 41},
				{TokenType: LITERAL, Value: "'surname with spaces'", Position: 43},
			},
		},
		{
			name:  "Test escaped chars",
			value: `SELECT * FROM ADDRESS_BOOK WHERE SURNAME = 'surname with spaces and \'quote\''`,
			expectedTokens: []Token{
				{TokenType: LITERAL, Value: "SELECT", Position: 0},
				{TokenType: LITERAL, Value: "*", Position: 7},
				{TokenType: LITERAL, Value: "FROM", Position: 9},
				{TokenType: LITERAL, Value: "ADDRESS_BOOK", Position: 14},
				{TokenType: LITERAL, Value: "WHERE", Position: 27},
				{TokenType: LITERAL, Value: "SURNAME", Position: 33},
				{TokenType: OP, Value: "=", Position: 41},
				{TokenType: LITERAL, Value: `'surname with spaces and \'quote\''`, Position: 43},
			},
		},
		{
			name:  "Test SQL with operators",
			value: `SELECT * FROM ADDRESS_BOOK WHERE SURNAME = 'Mouse' AND AGE > 3`,
			expectedTokens: []Token{
				{TokenType: LITERAL, Value: "SELECT", Position: 0},
				{TokenType: LITERAL, Value: "*", Position: 7},
				{TokenType: LITERAL, Value: "FROM", Position: 9},
				{TokenType: LITERAL, Value: "ADDRESS_BOOK", Position: 14},
				{TokenType: LITERAL, Value: "WHERE", Position: 27},
				{TokenType: LITERAL, Value: "SURNAME", Position: 33},
				{TokenType: OP, Value: "=", Position: 41},
				{TokenType: LITERAL, Value: "'Mouse'", Position: 43},
				{TokenType: LITERAL, Value: "AND", Position: 51},
				{TokenType: LITERAL, Value: "AGE", Position: 55},
				{TokenType: OP, Value: ">", Position: 59},
				{TokenType: LITERAL, Value: "3", Position: 61},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			scanner := NewSQLScanner()
			scanner.Init(tt.value)
			var allTokens []Token
			for scanner.Next() {
				allTokens = append(allTokens, *scanner.Token())
			}
			Expect(allTokens).To(Equal(tt.expectedTokens))
		})
	}
}
