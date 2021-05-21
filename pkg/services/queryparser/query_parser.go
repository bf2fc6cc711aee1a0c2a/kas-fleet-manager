package services

import (
	"fmt"
	"github.com/pkg/errors"
)

var validColumns = []string{"region", "name", "cloud_provider", "status", "owner"}

const BraceToken = 0
const OpToken = 1
const LogicalOpToken = 2
const ColumnToken = 3
const ValueToken = 4
const QuotedValueToken = 5
const StartToken = 99

const MaximumComplexity = 10

type checkUnbalancedBraces func() error

type DBQuery struct {
	Query  string
	Values []interface{}
}

type QueryParser interface {
	Parse(sql string) (*DBQuery, error)
}

type queryParser struct {
	dbqry DBQuery
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
		case BraceToken:
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
		case ValueToken:
			p.dbqry.Query = fmt.Sprintf("%s ?", p.dbqry.Query)
			p.dbqry.Values = append(p.dbqry.Values, token.value)
			//p.sqlstring = fmt.Sprintf("%s '%s'", p.sqlstring, strings.ReplaceAll(token.value, "'", `\'`))
			return nil
		case QuotedValueToken:
			// todo: unescape
			p.dbqry.Query = fmt.Sprintf("%s ?", p.dbqry.Query)
			p.dbqry.Values = append(p.dbqry.Values, token.value)

			//p.sqlstring = fmt.Sprintf("%s '%s'", p.sqlstring, token.value)
			return nil
		case LogicalOpToken:
			complexity++
			if complexity > MaximumComplexity {
				return errors.Errorf("maximum number of permitted joins (%d) exceeded", MaximumComplexity)
			}
			p.dbqry.Query = p.dbqry.Query + " " + token.value
			return nil
		case ColumnToken:
			if !contains(validColumns, token.value) {
				return fmt.Errorf("invalid column name: '%s'", token.value)
			}
			p.dbqry.Query = p.dbqry.Query + " " + token.value
			return nil
		default:
			p.dbqry.Query = p.dbqry.Query + " " + token.value
			return nil
		}
	}

	// TOKENS
	start := NewStateBuilder(StartToken).
		AcceptPattern(`^$`).
		Build()

	openBrace := NewStateBuilder(BraceToken).
		AcceptPattern(`^\(\s*$`).
		//Consumer(simpleConsumer).
		OnNewToken(onNewToken).
		Build()

	closedBrace := NewStateBuilder(BraceToken).
		AcceptPattern(`^\)\s*$`).
		Last().
		OnNewToken(onNewToken).
		Build()

	// column
	column := NewStateBuilder(ColumnToken).
		AcceptPattern(`^[A-Za-z][A-Za-z0-9_]*\s*$`).
		OnNewToken(onNewToken).
		Build()

	// value
	value := NewStateBuilder(ValueToken).AcceptPattern(`^[^ ^(^)]+\s*$`).OnNewToken(onNewToken).Last().Build()

	// quoted value
	openQuote := NewStateBuilder(ValueToken).AcceptPattern(`^'$`).Build()
	quotedValue := NewStateBuilder(QuotedValueToken).AcceptPattern(`^([^']|\\')+$`).OnNewToken(onNewToken).Build()
	closeQuote := NewStateBuilder(ValueToken).AcceptPattern(`^'\s*$`).Last().Build()

	// operators
	operatorEq := NewStateBuilder(OpToken).AcceptPattern(`^=\s*$`).OnNewToken(onNewToken).Build()
	operatorNotEq := NewStateBuilder(OpToken).AcceptPattern(`^[<>]{1,2}\s*$`).ValidatePattern(`^\s*<>\s*$`).OnNewToken(onNewToken).Build()
	operatorLike := NewStateBuilder(OpToken).AcceptPattern(`^[LlIiKkEe]{0,4}\s*$`).ValidatePattern(`^\s*[Ll][Ii][Kk][Ee]\s*$`).OnNewToken(onNewToken).Build()
	operatorAnd := NewStateBuilder(LogicalOpToken).AcceptPattern(`^[AaNnDd]{0,3}\s*$`).ValidatePattern(`^\s*[Aa][Nn][Dd]\s*$`).OnNewToken(onNewToken).Build()
	operatorOr := NewStateBuilder(LogicalOpToken).AcceptPattern(`^[OoRr]{0,2}\s*$`).ValidatePattern(`^\s*[Oo][Rr]\s*$`).OnNewToken(onNewToken).Build()

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

func (p *queryParser) Parse(sql string) (*DBQuery, error) {
	state, checkBalancedBraces := p.initStateMachine()
	for i, c := range sql {
		if next, err := state.parse(c); err != nil {
			return nil, errors.Errorf("[%d] error parsing the filter: %v", i+1, err)
		} else {
			state = next
		}
	}

	if err := state.eof(); err != nil {
		return nil, err
	}

	if err := checkBalancedBraces(); err != nil {
		return nil, err
	}
	return &p.dbqry, nil
}

func NewQueryParser() QueryParser {
	return &queryParser{dbqry: DBQuery{}}
}
