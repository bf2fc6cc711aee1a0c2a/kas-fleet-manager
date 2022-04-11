package stringscanner

var _ Scanner = &simpleStringScanner{}

const (
	ALPHA = iota
	DIGIT
	DECIMALPOINT
	SYMBOL
)

// simpleStringScanner splits the string into character. Each character will be a token.
type simpleStringScanner struct {
	value string
	pos   int
}

func (s *simpleStringScanner) Next() bool {
	if s.pos < len(s.value)-1 {
		s.pos++
		return true
	}
	return false
}

func (s *simpleStringScanner) Peek() (bool, *Token) {
	if s.pos < len(s.value)-1 {
		return true, updateTokenType(&Token{
			TokenType: 0,
			Value:     string(s.value[s.pos+1]),
			Position:  s.pos + 1,
		})
	}
	return false, nil
}

func (s *simpleStringScanner) Token() *Token {
	if s.pos < len(s.value) {
		return updateTokenType(&Token{
			TokenType: 0,
			Value:     string(s.value[s.pos]),
			Position:  s.pos,
		})
	}

	panic("No tokens available")
}

func (s *simpleStringScanner) Init(value string) {
	s.pos = -1
	s.value = value
}

func updateTokenType(token *Token) *Token {
	runeAry := []rune(token.Value)
	c := runeAry[0]

	switch true {
	case c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z':
		token.TokenType = ALPHA
	case c >= '0' && c <= '9':
		token.TokenType = DIGIT
	case c == '.':
		token.TokenType = DECIMALPOINT
	default:
		token.TokenType = SYMBOL
	}
	return token
}

func NewSimpleScanner() Scanner {
	return &simpleStringScanner{}
}
