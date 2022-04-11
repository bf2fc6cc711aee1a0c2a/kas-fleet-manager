package stringscanner

// Token - The Token
type Token struct {
	// TokenType - This depends on the Scanner implementation and is used to give info about the type of token found
	TokenType int
	// Value - The value of the current token
	Value string
	// Position - Indicates the position (0 based) where the Token has been found
	Position int
}

// Scanner is used to split a string into Tokens
type Scanner interface {
	// Next - Move to the next Token. Return false if no next Token is available
	Next() bool
	// Peek - Look at the next Token without moving. Return false if no next Token is available
	Peek() (bool, *Token)
	// Token - Return the current Token Value. Panics if current Position is invalid.
	Token() *Token
	// Init - Initialise the scanner with the given string
	Init(s string)
}
