package services

import (
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

type Token struct {
	tokenType int
	value     string
}

type state struct {
	tokenType       int
	value           string
	acceptPattern   string
	validatePattern string

	last bool
	next []State

	valueValidator func(token *Token) error
	onNewToken     func(token *Token) error
	consumer       func(token *Token)
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

	// If a custom validator is present, this is the right time to run it
	if s.valueValidator != nil {
		if err := s.valueValidator(&Token{
			tokenType: s.tokenType,
			value:     strings.Trim(s.value, " "),
		}); err != nil {
			return nil, err
		}
	}

	if s.onNewToken != nil {
		if err := s.onNewToken(&Token{
			tokenType: s.tokenType,
			value:     strings.Trim(s.value, " "),
		}); err != nil {
			return nil, err
		}
	}

	if s.consumer != nil {
		s.consumer(&Token{
			tokenType: s.tokenType,
			value:     strings.Trim(s.value, " "),
		})
	}

	// Current state is valid. We don't need this value anymore
	s.value = ""

	// Move to next state
	for _, n := range s.next {
		if n.accept(c) {
			return n.parse(c)
		}
	}

	return nil, errors.Errorf("unexpected token '%s'", string(c))
}

func (s *state) eof() error {
	// EOF has been reached. Check if the current token can be the last one
	if !s.last {
		return errors.Errorf(`EOF encountered while parsing string (last token found: "%s")`, s.value)
	}

	// Validate the last token
	if s.valueValidator != nil {
		if err := s.valueValidator(&Token{
			tokenType: s.tokenType,
			value:     s.value,
		}); err != nil {
			return err
		}
	}

	if s.onNewToken != nil {
		if err := s.onNewToken(&Token{
			tokenType: s.tokenType,
			value:     strings.Trim(s.value, " "),
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *state) addNextState(next State) {
	s.next = append(s.next, next)
}

// StateBuilder
type StateBuilder interface {
	AcceptPattern(acceptRegex string) StateBuilder
	ValidatePattern(validatePattern string) StateBuilder
	ValueValidator(validator func(token *Token) error) StateBuilder
	OnNewToken(handler func(token *Token) error) StateBuilder
	Consumer(consumer func(token *Token)) StateBuilder
	Last() StateBuilder
	Build() State
}

type stateBuilder struct {
	s *state
}

var _ StateBuilder = &stateBuilder{}

func (sb *stateBuilder) AcceptPattern(acceptPattern string) StateBuilder {
	sb.s.acceptPattern = acceptPattern
	return sb
}

func (sb *stateBuilder) ValidatePattern(validatePattern string) StateBuilder {
	sb.s.validatePattern = validatePattern
	return sb
}

func (sb *stateBuilder) Last() StateBuilder {
	sb.s.last = true
	return sb
}

func (sb *stateBuilder) ValueValidator(validator func(token *Token) error) StateBuilder {
	sb.s.valueValidator = validator
	return sb
}

func (sb *stateBuilder) OnNewToken(handler func(token *Token) error) StateBuilder {
	sb.s.onNewToken = handler
	return sb
}

func (sb *stateBuilder) Consumer(consumer func(token *Token)) StateBuilder {
	sb.s.consumer = consumer
	return sb
}

func (sb *stateBuilder) Build() State {
	if sb.s.validatePattern == "" {
		sb.s.validatePattern = sb.s.acceptPattern
	}
	return sb.s
}

func NewStateBuilder(tokenType int) StateBuilder {
	return &stateBuilder{s: &state{
		last:      false,
		tokenType: tokenType,
	}}
}
