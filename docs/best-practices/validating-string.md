# Validating strings and grammars
If you need to validate complex string, instead of writing custom, code, try using the provided state machine utility.

Suppose you need to validate a size, and the size can be expressed by a number followed by the unit of measure.
The unit of measure can be one of (case insensitive):
* GB/Gib
* MB/MiB
* KB/Kib
* B

Valid examples will be:
* 100MB
* 100mb
* 100Gib
* 1000GIB
* .20kb

## Configure the grammar
To parse the string, we will need to configure the grammar. In our case, the tokens will be:
* DIGIT [0-9]
* DECIMALPOINT .
* DECIMAL [0-9]
* G [Gg]
* M [Mm]
* K [Kk]
* I [Ii]
* B [Bb]

This can be easily done using the `Grammar` object:
```go
grammar := services.Grammar{
    Tokens: []services.TokenDefinition{
        {Name: "DIGIT", AcceptPattern: "[0-9]"},
        {Name: "DECIMALPOINT", AcceptPattern: `\.`},
        {Name: "DECIMAL", AcceptPattern: "[0-9]"},
        {Name: "G", AcceptPattern: "(?i)G"},
        {Name: "M", AcceptPattern: "(?i)M"},
        {Name: "K", AcceptPattern: "(?i)K"},
        {Name: "I", AcceptPattern: "(?i)I"},
        {Name: "B", AcceptPattern: "(?i)B"},
    },
}
```
Now that we defined the grammar, we need to define the valid transitions:
```
START -> DIGIT | DECIMALPOINT
DIGIT -> DIGIT | DECIMALPOINT | G | M | K | B
DECIMALPOINT -> DECIMAL
DECIMAL -> DECIMAL | G | M | K | B
G -> I | B
M -> I | B
K -> I | B
I -> B
B -> END
```
Again, those transitions can be easily defined using the `Grammar` object:
```go
grammar := services.Grammar{
    Tokens: []services.TokenDefinition{
        {Name: "DIGIT", AcceptPattern: "[0-9]"},
        {Name: "DECIMALPOINT", AcceptPattern: `\.`},
        {Name: "DECIMAL", AcceptPattern: "[0-9]"},
        {Name: "G", AcceptPattern: "(?i)G"},
        {Name: "M", AcceptPattern: "(?i)M"},
        {Name: "K", AcceptPattern: "(?i)K"},
        {Name: "I", AcceptPattern: "(?i)I"},
        {Name: "B", AcceptPattern: "(?i)B"},
    },
    Transitions: []services.TransitionDefinition{
        {TokenName: services.StartState, ValidTransitions: []string{"DIGIT", "DECIMALPOINT"}},
        {TokenName: "DIGIT", ValidTransitions: []string{"DIGIT", "DECIMALPOINT", "G", "M", "K", "B"}},
        {TokenName: "DECIMALPOINT", ValidTransitions: []string{"DECIMAL"}},
        {TokenName: "DECIMAL", ValidTransitions: []string{"DECIMAL", "G", "M", "K", "B"}},
        {TokenName: "G", ValidTransitions: []string{"I", "B"}},
        {TokenName: "M", ValidTransitions: []string{"I", "B"}},
        {TokenName: "K", ValidTransitions: []string{"I", "B"}},
        {TokenName: "I", ValidTransitions: []string{"B"}},
        {TokenName: "B", ValidTransitions: []string{services.EndState}},
    },
}
```
## Validate the string

Now that we have the grammar, we can parse and validate the string. To do that, we need to use a `scanner`: a `scanner` is an object that split the received `string` into `tokens`. Since our string is very simple, we can use the `SimpleScanner` that returns a `token` for each char of the string.

```go
func IsStringValid(grammar *Grammar, value string) (bool, error) {
    startState := services.NewStateMachineBuilder(&q.grammar).Build()
    scanner := services.NewSimpleScanner()
    scanner.Init(value)

    state := startState
    for scanner.Next() {
        if next, err := state.Parse(scanner.Token().Value); err != nil {
            return false, fmt.Errorf("[%d] error parsing the filter: %v", scanner.Token().Position+1, err)
        } else {
            state = next
        }
    }
    
    if err := state.Eof(); err != nil {
        return false, err
    }
    
    return true, nil
}
```

## Creating a parser object
All the previous code can be condensed inside a *parser object* so that using it will become very easy

```go
type QuantityParser interface {
    IsValid(value string) (bool, error)
}

type quantityParser struct {
    grammar services.Grammar
}

func (q *quantityParser) IsValid(value string) (bool, error) {
    startState := services.NewStateMachineBuilder(&q.grammar).Build()
    scanner := services.NewSimpleScanner()
    scanner.Init(value)
    
    state := startState
    for scanner.Next() {
        if next, err := state.Parse(scanner.Token().Value); err != nil {
            return false, fmt.Errorf("[%d] error parsing the filter: %v", scanner.Token().Position+1, err)
        } else {
            state = next
        }
    }
    
    if err := state.Eof(); err != nil {
        return false, err
    }
    
    return true, nil
}

func NewQuantityParser() QuantityParser {
    return &quantityParser{
        grammar: services.Grammar{
        Tokens: []services.TokenDefinition{
            {Name: "DIGIT", AcceptPattern: "[0-9]"},
            {Name: "DECIMALPOINT", AcceptPattern: `\.`},
            {Name: "DECIMAL", AcceptPattern: "[0-9]"},
            {Name: "G", AcceptPattern: "(?i)G"},
            {Name: "M", AcceptPattern: "(?i)M"},
            {Name: "K", AcceptPattern: "(?i)K"},
            {Name: "I", AcceptPattern: "(?i)I"},
            {Name: "B", AcceptPattern: "(?i)B"},
        },
        Transitions: []services.TransitionDefinition{
            {TokenName: services.StartState, ValidTransitions: []string{"DIGIT", "DECIMALPOINT"}},
            {TokenName: "DIGIT", ValidTransitions: []string{"DIGIT", "DECIMALPOINT", "G", "M", "K", "B"}},
            {TokenName: "DECIMALPOINT", ValidTransitions: []string{"DECIMAL"}},
            {TokenName: "DECIMAL", ValidTransitions: []string{"DECIMAL", "G", "M", "K", "B"}},
            {TokenName: "G", ValidTransitions: []string{"I", "B"}},
            {TokenName: "M", ValidTransitions: []string{"I", "B"}},
            {TokenName: "K", ValidTransitions: []string{"I", "B"}},
            {TokenName: "I", ValidTransitions: []string{"B"}},
            {TokenName: "B", ValidTransitions: []string{services.EndState}},
        },
    },
    }
}
```
Now, validating the string will be very easy:
```go
func ValidateQuantity(s string) {
    qp := NewQuantityParser()
    if valid, err := qp.IsValid(s); valid {
        fmt.Println(s, "is valid")
    } else {
        fmt.Println(s, "is not valid:", err)
    }	
}
```
Calling the function with an invalid string will produce a detailed error.
For example, calling it with the value "100.2.3mb" will result in:
```
100.2.3mb is not valid: [6] error parsing the filter: Unexpected Token `.`
```
where `[6]` is the position in the string of the invalid token.
Same if we pass "100.3MOB":
```
100.3MOB is not valid: [7] error parsing the filter: Unexpected Token `O`
```
The parser will even detect if something is missing at the end. For example, calling it with the value "100.3" will result in:
```
100.3 is not valid: premature EOF encountered
```
## Provided Scanners
We currently provide 2 scanners:
* **[SimpleScanner](../../pkg/shared/utils/stringscanner/simple_string_scanner.go)**: this scanner produces a token for each char of the string.
* **SQLScanner**: this scanner splits the given string according to SQL rules (LITERALS, ROUND_BRACES, QUOTED_LITERALS, etc). An example usage of this scanner can be seen in [query_parser.go](../../pkg/services/queryparser/query_parser.go)
