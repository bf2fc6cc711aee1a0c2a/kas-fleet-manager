package services

import (
	"fmt"
	"github.com/pkg/errors"
	"strings"
)

var validColumns = []string{"column_provider", "name", "owner", "region", "status"}

const BRACE_TOKEN = 0
const OP_TOKEN = 1
const LOGICAL_OP_TOKEN = 2
const COLUMN_TOKEN = 3
const VALUE_TOKEN = 4
const QUOTED_VALUE_TOKEN = 5
const START_TOKEN = 99
const MAXIMUM_COMPLEXITY = 10

type checkUnbalancedBraces func() error

type QueryParser interface {
	Parse(sql string) error
	GetQueryString() string
}

type queryParser struct {
	sqlstring string
}

var _ QueryParser = &queryParser{}

// initStateMachine
// This will be our grammar (each token will eat the spaces after the token itself):
// Tokens:
// OPEN_BRACE       =  ^\(\s*$
// CLOSED_BRACE     = ^\)\s*$
// COLUMN -         = ^[A-Za-z][A-Za-z0-9_]*\s*$
// VALUE            = ^[^ ^(^)]+\s*$
// QUOTED_VALUE     = OPEN_QUOTE Q_VALUE CLOSED_QUOTE
//     OPEN_QUOTE   = ^'$
//     Q_VALUE      = ^([^']|\\')+$   - any character but `'` unless escaped (\')
//     CLOSED_QUOTE = ^'$
// EQ               = ^=\s*$
// NOT_EQ           = ^\s*<>\s*$
// LIKE             = ^\s*[Ll][Ii][Kk][Ee]\s*$
// AND              = ^\s*[Aa][Nn][Dd]\s*$
// OR               = ^\s*[Oo][Rr]\s*$
//
// VALID TRANSITIONS:
// START        -> COLUMN | OPEN_BRACE
// OPEN_BRACE   -> OPEN_BRACE | COLUMN
// COLUMN       -> EQ | NOT_EQ | LIKE
// EQ           -> VALUE | QUOTED_VALUE
// NOT_EQ       -> VALUE | QUOTED_VALUE
// LIKE         -> VALUE | QUOTED_VALUE
// VALUE        -> OR | AND | CLOSED_BRACE | [END]
// QUOTED_VALUE -> OR | AND | CLOSED_BRACE | [END]
// CLOSED_BRACE -> OR | AND | CLOSED_BRACE | [END]
// AND          -> COLUMN | OPEN_BRACE
// OR           -> COLUMN | OPEN_BRACE
func (p *queryParser) initStateMachine() (State, checkUnbalancedBraces) {
	// This variable counts the open braces
	braces := 0
	complexity := 0

	contains := func(s []string, value string) bool {
		for _, item := range s {
			if item == value {
				return true
			}
		}
		return false
	}

	onNewToken := func(token *Token) error {
		switch token.tokenType {
		case BRACE_TOKEN:
			if token.value == "(" {
				braces++
				return nil
			} else {
				braces--
				if braces < 0 {
					return errors.Errorf("unexpected ')'")
				}
				return nil
			}
		case VALUE_TOKEN:
			p.sqlstring = fmt.Sprintf("%s '%s'", p.sqlstring, strings.ReplaceAll(token.value, "'", `\'`))
			return nil
		case QUOTED_VALUE_TOKEN:
			p.sqlstring = fmt.Sprintf("%s '%s'", p.sqlstring, token.value)
			return nil
		case LOGICAL_OP_TOKEN:
			complexity++
			if complexity > MAXIMUM_COMPLEXITY {
				return errors.Errorf("maximum number of permitted joins (%d) exceeded", MAXIMUM_COMPLEXITY)
			}
			p.sqlstring = p.sqlstring + " " + token.value
			return nil
		case COLUMN_TOKEN:
			if !contains(validColumns, token.value) {
				return fmt.Errorf("invalid column name: '%s'", token.value)
			}
			p.sqlstring = p.sqlstring + " " + token.value
			return nil
		default:
			p.sqlstring = p.sqlstring + " " + token.value
			return nil
		}
	}

	// TOKENS
	start := NewStateBuilder(START_TOKEN).
		AcceptPattern(`^$`).
		Build()

	openBrace := NewStateBuilder(BRACE_TOKEN).
		AcceptPattern(`^\(\s*$`).
		//Consumer(simpleConsumer).
		OnNewToken(onNewToken).
		Build()

	closedBrace := NewStateBuilder(BRACE_TOKEN).
		AcceptPattern(`^\)\s*$`).
		Last().
		OnNewToken(onNewToken).
		Build()

	// column
	column := NewStateBuilder(COLUMN_TOKEN).
		AcceptPattern(`^[A-Za-z][A-Za-z0-9_]*\s*$`).
		OnNewToken(onNewToken).
		Build()

	// value
	value := NewStateBuilder(VALUE_TOKEN).AcceptPattern(`^[^ ^(^)]+\s*$`).OnNewToken(onNewToken).Last().Build()

	// quoted value
	openQuote := NewStateBuilder(VALUE_TOKEN).AcceptPattern(`^'$`).Build()
	quotedValue := NewStateBuilder(QUOTED_VALUE_TOKEN).AcceptPattern(`^([^']|\\')+$`).OnNewToken(onNewToken).Build()
	closeQuote := NewStateBuilder(VALUE_TOKEN).AcceptPattern(`^'\s*$`).Last().Build()

	// operators
	operatorEq := NewStateBuilder(OP_TOKEN).AcceptPattern(`^=\s*$`).OnNewToken(onNewToken).Build()
	operatorNotEq := NewStateBuilder(OP_TOKEN).AcceptPattern(`^[<>]{1,2}\s*$`).ValidatePattern(`^\s*<>\s*$`).OnNewToken(onNewToken).Build()
	operatorLike := NewStateBuilder(OP_TOKEN).AcceptPattern(`^[LlIiKkEe]{0,4}\s*$`).ValidatePattern(`^\s*[Ll][Ii][Kk][Ee]\s*$`).OnNewToken(onNewToken).Build()
	operatorAnd := NewStateBuilder(LOGICAL_OP_TOKEN).AcceptPattern(`^[AaNnDd]{0,3}\s*$`).ValidatePattern(`^\s*[Aa][Nn][Dd]\s*$`).OnNewToken(onNewToken).Build()
	operatorOr := NewStateBuilder(LOGICAL_OP_TOKEN).AcceptPattern(`^[OoRr]{0,2}\s*$`).ValidatePattern(`^\s*[Oo][Rr]\s*$`).OnNewToken(onNewToken).Build()

	// state transitions

	start.addNextState(column)
	start.addNextState(openBrace)

	openBrace.addNextState(column)
	openBrace.addNextState(openBrace)

	column.addNextState(operatorEq)
	column.addNextState(operatorNotEq)
	column.addNextState(operatorLike)

	operatorEq.addNextState(openQuote)
	operatorEq.addNextState(value)

	operatorNotEq.addNextState(openQuote)
	operatorNotEq.addNextState(value)

	operatorLike.addNextState(openQuote)
	operatorLike.addNextState(value)

	openQuote.addNextState(quotedValue)
	openQuote.addNextState(closeQuote)
	quotedValue.addNextState(closeQuote)

	value.addNextState(operatorOr)
	value.addNextState(operatorAnd)
	value.addNextState(closedBrace)

	closeQuote.addNextState(operatorOr)
	closeQuote.addNextState(operatorAnd)
	closeQuote.addNextState(closedBrace)

	closedBrace.addNextState(operatorOr)
	closedBrace.addNextState(operatorAnd)
	closedBrace.addNextState(closedBrace)

	operatorAnd.addNextState(column)
	operatorAnd.addNextState(openBrace)
	operatorOr.addNextState(column)
	operatorOr.addNextState(openBrace)

	return start, func() error {
		if braces > 0 {
			return fmt.Errorf("EOF while searching for closing brace ')'")
		}

		return nil
	}
}

func (p *queryParser) Parse(sql string) error {
	state, checkBalancedBraces := p.initStateMachine()
	for i, c := range sql {
		if next, err := state.parse(c); err != nil {
			return errors.Errorf("[%d] error parsing the filter: %v", i+1, err)
		} else {
			state = next
		}
	}

	if err := state.eof(); err != nil {
		return err
	}

	if err := checkBalancedBraces(); err != nil {
		return err
	}
	return nil
}

func (p *queryParser) GetQueryString() string {
	return p.sqlstring
}

func NewQueryParser() QueryParser {
	return &queryParser{}
}
