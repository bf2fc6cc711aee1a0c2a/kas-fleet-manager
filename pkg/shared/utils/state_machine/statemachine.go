package state_machine

import (
	"fmt"
	"github.com/pkg/errors"
	"regexp"
	"strings"
)

// State - represent a single state in the current state machine
type State interface {
	// Move - Analysing the passed in value, decides what will be the next state. Returns the next state or error received value is invalid
	Move(tok string) (State, error)
	// Eof - returns whether the current state is a valid END STATE
	Eof() bool
}

var _ State = &state{}

// ParsedToken - Structure sent to the callback everytime a new ParsedToken is parsed
type ParsedToken struct {
	// Name - the name of this Token
	Name string
	// Family - the Family assigned to this Token. Used when something needs to be done for each Token of the same Family
	Family string
	// Value - the Value of the Token
	Value string
}

// state - internal structure defining a state
type state struct {
	// tokenName - name of the token managed by this state
	tokenName string

	// family - family of the token managed by this state
	family string
	// acceptPattern - pattern used to decide if the current character can be accepted as part of this state Value
	acceptPattern string

	// last - this is set to true if this state can be the last state (just before the EOF)
	last bool

	// isEof - used internally to define the END state. Not to be used by other states
	isEof bool

	// next - the list of valid transitions from this state
	next []State

	// onNewToken - sets the handler to be invoked when a Token has been successfully parsed
	onNewToken func(token *ParsedToken) error
}

func newStartState() State {
	return &state{
		tokenName:     "START",
		acceptPattern: `^$`,
	}
}

func newEndState() State {
	return &state{
		tokenName: "END",
		isEof:     true,
	}
}

func (s *state) accept(value string) bool {
	matched, _ := regexp.Match(s.acceptPattern, []byte(value))
	return matched
}

func (s *state) Move(value string) (State, error) {
	for _, next := range s.next {
		if next.(*state).accept(value) {
			// valid Value
			if next.(*state).onNewToken != nil {
				if err := next.(*state).onNewToken(&ParsedToken{
					Name:   next.(*state).tokenName,
					Family: next.(*state).family,
					Value:  value,
				}); err != nil {
					return nil, err
				}
			}
			return next, nil
		}
	}

	return nil, errors.Errorf("Unexpected Token `%s`", value)
}

// Eof - this function must be called when the whole string has been parsed to check if the current state is a valid eof state
func (s *state) Eof() bool {
	// EOF has been reached. Check if the current Token can be the last one
	return s.last
}

func (s *state) addNextState(next State) {
	n := next.(*state)
	if n.isEof {
		// if the passed in next state is an Eof state, means this is a valid 'last' state
		// Just save the info and discard the 'next' state
		s.last = true
	} else {
		s.next = append(s.next, next)
	}
}

// StateBuilder - builder of State objects
type StateBuilder interface {
	Family(family string) StateBuilder
	AcceptPattern(acceptRegex string) StateBuilder
	OnNewToken(handler func(token *ParsedToken) error) StateBuilder
	Build() State
}

type stateBuilder struct {
	s *state
}

var _ StateBuilder = &stateBuilder{}

func (sb *stateBuilder) Family(family string) StateBuilder {
	sb.s.family = family
	return sb
}

func (sb *stateBuilder) AcceptPattern(acceptPattern string) StateBuilder {
	sb.s.acceptPattern = acceptPattern
	return sb
}

func (sb *stateBuilder) OnNewToken(handler func(token *ParsedToken) error) StateBuilder {
	sb.s.onNewToken = handler
	return sb
}

func (sb *stateBuilder) Build() State {
	if !strings.HasPrefix(sb.s.acceptPattern, `^`) {
		sb.s.acceptPattern = fmt.Sprintf(`^%s`, sb.s.acceptPattern)
	}
	if !strings.HasSuffix(sb.s.acceptPattern, `$`) {
		sb.s.acceptPattern = fmt.Sprintf(`%s$`, sb.s.acceptPattern)
	}
	return sb.s
}

func NewStateBuilder(tokenName string) StateBuilder {
	return &stateBuilder{s: &state{
		last:      false,
		tokenName: tokenName,
	}}
}
