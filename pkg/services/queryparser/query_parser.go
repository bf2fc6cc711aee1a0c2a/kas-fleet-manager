package services

import (
	"fmt"
	"github.com/pkg/errors"
	"strings"
)

var validColumns = []string{"region", "name", "cloud_provider", "status", "owner"}

const (
	BraceTokenFamily       = "BRACE"
	OpTokenFamily          = "OP"
	LogicalOpTokenFamily   = "LOGICAL"
	ColumnTokenFamily      = "COLUMN"
	ValueTokenFamily       = "VALUE"
	QuoteTokenFamily       = "QUOTE"
	QuotedValueTokenFamily = "QUOTED"

	OpenBrace   = "OPEN_BRACE"
	ClosedBrace = "CLOSED_BRACE"
	Column      = "COLUMN"
	Value       = "VALUE"
	OpenQuote   = "OPEN_QUOTE"
	QuotedValue = "QUOTED_VALUE"
	CloseQuote  = "CLOSE_QUOTE"
	Eq          = "EQ"
	NotEq       = "NOT_EQ"
	LikeState   = "LIKE"
	AndState    = "AND"
	OrState     = "OR"
)
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
// OPEN_BRACE       = (
// CLOSED_BRACE     = )
// COLUMN -         = [A-Za-z][A-Za-z0-9_]*
// VALUE            = [^ ^(^)]+
// QUOTED_VALUE     = OPEN_QUOTE Q_VALUE CLOSED_QUOTE
//     OPEN_QUOTE   = '
//     Q_VALUE      = ([^']|\\')+   - any character but `'` unless escaped (\')
//     CLOSED_QUOTE = '
// EQ               = =
// NOT_EQ           = <>
// LIKE             = [Ll][Ii][Kk][Ee]
// AND              = [Aa][Nn][Dd]
// OR               = [Oo][Rr]
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
		switch token.family {
		case BraceTokenFamily:
			if token.value == "(" {
				braces++
			} else {
				braces--
				if braces < 0 {
					return errors.Errorf("unexpected ')'")
				}
			}

			p.dbqry.Query = p.dbqry.Query + token.value
			return nil
		case ValueTokenFamily:
			p.dbqry.Query = fmt.Sprintf("%s ?", p.dbqry.Query)
			p.dbqry.Values = append(p.dbqry.Values, token.value)
			return nil
		case QuotedValueTokenFamily:
			p.dbqry.Query = fmt.Sprintf("%s ?", p.dbqry.Query)
			// unescape
			p.dbqry.Values = append(p.dbqry.Values, strings.ReplaceAll(token.value, `\'`, "'"))

			return nil
		case LogicalOpTokenFamily:
			complexity++
			if complexity > MaximumComplexity {
				return errors.Errorf("maximum number of permitted joins (%d) exceeded", MaximumComplexity)
			}
			p.dbqry.Query = p.dbqry.Query + " " + token.value + " "
			return nil
		case ColumnTokenFamily:
			// we want column names to be lowercase
			columnName := strings.ToLower(token.value)
			if !contains(validColumns, columnName) {
				return fmt.Errorf("invalid column name: '%s'", token.value)
			}
			p.dbqry.Query = p.dbqry.Query + columnName
			return nil
		case QuoteTokenFamily:
			// ignore: we don't need quotes in the parsed result
			return nil
		default:
			p.dbqry.Query = p.dbqry.Query + " " + token.value
			return nil
		}
	}

	grammar := Grammar{
		Tokens: []TokenDefinition{
			{Name: OpenBrace, Family: BraceTokenFamily, AcceptPattern: `\(`},
			{Name: ClosedBrace, Family: BraceTokenFamily, AcceptPattern: `\)`},
			{Name: Column, Family: ColumnTokenFamily, AcceptPattern: `[A-Za-z][A-Za-z0-9_]*`},
			{Name: Value, Family: ValueTokenFamily, AcceptPattern: `[^ ^(^)]+`},
			/// START - quoted value
			{Name: OpenQuote, Family: QuoteTokenFamily, AcceptPattern: `'`, EvaluateSpaces: true},
			{Name: QuotedValue, Family: QuotedValueTokenFamily, AcceptPattern: `([^']|\\')+`, EvaluateSpaces: true},
			{Name: CloseQuote, Family: QuoteTokenFamily, AcceptPattern: `'`},
			/// END - quoted value
			{Name: Eq, Family: OpTokenFamily, AcceptPattern: `=`},
			{Name: NotEq, Family: OpTokenFamily, AcceptPattern: `[<>]{1,2}`, ValidatePattern: `<>`},
			{Name: LikeState, Family: OpTokenFamily, AcceptPattern: `[LlIiKkEe]{0,4}`, ValidatePattern: `[Ll][Ii][Kk][Ee]`},
			{Name: AndState, Family: LogicalOpTokenFamily, AcceptPattern: `[AaNnDd]{0,3}`, ValidatePattern: `[Aa][Nn][Dd]`},
			{Name: OrState, Family: LogicalOpTokenFamily, AcceptPattern: `[OoRr]{0,2}`, ValidatePattern: `[Oo][Rr]`},
		},
		Transitions: []TransitionDefinition{
			{TokenName: StartState, ValidTransitions: []string{Column, OpenBrace}},
			{TokenName: OpenBrace, ValidTransitions: []string{Column, OpenBrace}},
			{TokenName: Column, ValidTransitions: []string{Eq, NotEq, LikeState}},
			{TokenName: Eq, ValidTransitions: []string{OpenQuote, Value}},
			{TokenName: NotEq, ValidTransitions: []string{OpenQuote, Value}},
			{TokenName: LikeState, ValidTransitions: []string{OpenQuote, Value}},
			{TokenName: OpenQuote, ValidTransitions: []string{QuotedValue, CloseQuote}},
			{TokenName: OpenQuote, ValidTransitions: []string{QuotedValue, CloseQuote}},
			{TokenName: QuotedValue, ValidTransitions: []string{CloseQuote}},
			{TokenName: CloseQuote, ValidTransitions: []string{OrState, AndState, ClosedBrace, EndState}},
			{TokenName: Value, ValidTransitions: []string{OrState, AndState, ClosedBrace, EndState}},
			{TokenName: ClosedBrace, ValidTransitions: []string{OrState, AndState, ClosedBrace, EndState}},
			{TokenName: AndState, ValidTransitions: []string{Column, OpenBrace}},
			{TokenName: OrState, ValidTransitions: []string{Column, OpenBrace}},
		},
	}

	start := NewStateMachineBuilder(&grammar).
		OnNewToken(onNewToken).
		Build()

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
	p.dbqry.Query = strings.Trim(p.dbqry.Query, " ")
	return &p.dbqry, nil
}

func NewQueryParser() QueryParser {
	return &queryParser{dbqry: DBQuery{}}
}
