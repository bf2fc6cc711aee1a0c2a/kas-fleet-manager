package services

import (
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"strings"
)

type State interface {
	accept(c int32) bool
	parse(c int32) (State, error)
	eof() error

	addNextState(s State)
}

var _ State = &state{}

// Token - Structure sent to the callback everytime a new Token is parsed
type Token struct {
	// tokenName - the name of this token
	tokenName string
	// family - the family assigned to this token. Used when something needs to be done for each token of the same family
	family string
	// value - the value of the token
	value string
}

// state - internal structure defining a state
type state struct {
	tokenName string
	family    string
	value     string
	// acceptPattern - pattern used to decide if the current character can be accepted as part of this token value
	acceptPattern string
	// validatePattern - pattern used to validate the token value when the token parsing is finished
	validatePattern string

	// last - this is set to true if this token can be the last token (just before the EOF)
	last bool

	// isEof - used internally to define the END token. Not to be used by other tokens
	isEof bool

	// next - the list of valid transitions from this state
	next []State

	// ignoreSurroundingSpaces - true if spaces following this token must be eaten and discarded
	ignoreSurroundingSpaces bool
	// onNewToken - handler to be invoked when a token has been successfully parsed
	onNewToken func(token *Token) error
}

func NewStartState() State {
	return &state{
		acceptPattern: `^$`,
	}
}

func NewEndState() State {
	return &state{
		isEof: true,
	}
}

func (s *state) accept(c int32) bool {
	tmp := s.value + string(c)
	matched, _ := regexp.Match(s.acceptPattern, []byte(tmp))
	return matched
}

func (s *state) parse(c int32) (State, error) {
	// If the current character is valid for the current state, add it to the current state value
	if s.accept(c) {
		s.value = s.value + string(c)
		return s, nil
	}

	// Before passing to the next state, check that the finished value of the current state is valid
	if matched, _ := regexp.Match(s.validatePattern, []byte(s.value)); !matched {
		return nil, errors.Errorf("invalid expression '%s%s'", s.value, string(c))
	}

	// A new token has been successfully parsed: invoke the onNewToken handler if present
	if s.onNewToken != nil {
		if err := s.onNewToken(&Token{
			tokenName: s.tokenName,
			family:    s.family,
			value:     s.getValue(),
		}); err != nil {
			return nil, err
		}
	}

	// We sent the token value to the handler. We can reset now the value so that this state can be reused.
	s.value = ""

	// Move to next state
	for _, n := range s.next {
		if n.accept(c) {
			return n.parse(c)
		}
	}

	return nil, errors.Errorf("unexpected token '%s'", string(c))
}

func (s *state) getValue() string {
	if s.ignoreSurroundingSpaces {
		return strings.Trim(s.value, " ")
	} else {
		return s.value
	}
}

// eof - this function must be called when the whole string has been parsed to check if the current state is a valid eof state
func (s *state) eof() error {
	// EOF has been reached. Check if the current token can be the last one
	if !s.last {
		return errors.Errorf(`EOF encountered while parsing string (last token found: "%s")`, s.value)
	}

	// Pass the last token to the onNewTokenHandler
	if s.onNewToken != nil {
		if err := s.onNewToken(&Token{
			tokenName: s.tokenName,
			family:    s.family,
			value:     s.getValue(),
		}); err != nil {
			return err
		}
	}

	// reset the token value
	s.value = ""
	return nil
}

func (s *state) addNextState(next State) {
	n := next.(*state)
	if n.isEof {
		// if the passed in next state is an eof state, means this is a valid 'last' state
		// Just notify that and discard the 'next' state
		s.last = true
	} else {
		s.next = append(s.next, next)
	}
}

// StateBuilder - builder of State objects
type StateBuilder interface {
	Family(family string) StateBuilder
	AcceptPattern(acceptRegex string) StateBuilder
	ValidatePattern(validatePattern string) StateBuilder
	DontIgnoreSurroundingSpaces() StateBuilder
	OnNewToken(handler func(token *Token) error) StateBuilder
	Build() State
}

type stateBuilder struct {
	s *state
}

var _ StateBuilder = &stateBuilder{}

func (sb *stateBuilder) DontIgnoreSurroundingSpaces() StateBuilder {
	sb.s.ignoreSurroundingSpaces = false
	return sb
}

func (sb *stateBuilder) Family(family string) StateBuilder {
	sb.s.family = family
	return sb
}

func (sb *stateBuilder) AcceptPattern(acceptPattern string) StateBuilder {
	sb.s.acceptPattern = acceptPattern
	return sb
}

func (sb *stateBuilder) ValidatePattern(validatePattern string) StateBuilder {
	sb.s.validatePattern = validatePattern
	return sb
}

func (sb *stateBuilder) OnNewToken(handler func(token *Token) error) StateBuilder {
	sb.s.onNewToken = handler
	return sb
}

func (sb *stateBuilder) Build() State {
	if sb.s.validatePattern == "" {
		sb.s.validatePattern = sb.s.acceptPattern
	}

	if sb.s.ignoreSurroundingSpaces {
		sb.s.acceptPattern = fmt.Sprintf(`^%s\s*$`, sb.s.acceptPattern)
		sb.s.validatePattern = fmt.Sprintf(`^%s\s*$`, sb.s.validatePattern)
	} else {
		sb.s.acceptPattern = fmt.Sprintf(`^%s$`, sb.s.acceptPattern)
		sb.s.validatePattern = fmt.Sprintf(`^%s$`, sb.s.validatePattern)
	}
	return sb.s
}

func NewStateBuilder(tokenName string) StateBuilder {
	return &stateBuilder{s: &state{
		last:                    false,
		ignoreSurroundingSpaces: true,
		tokenName:               tokenName,
	}}
}
