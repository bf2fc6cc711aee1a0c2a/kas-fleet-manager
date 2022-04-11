package state_machine

const (
	StartState = "__$$_START_$$__"
	EndState   = "__$$_END_$$__"
)

type StateDefinition struct {
	Name          string
	Family        string
	AcceptPattern string
	OnNewToken    func(token *ParsedToken) error
}

type StateMachineDefinition struct {
	States      []StateDefinition
	Transitions []TransitionDefinition
}

type TransitionDefinition struct {
	StateName        string
	ValidTransitions []string
}

type NewTokenHandler func(token *ParsedToken) error

var _ StateMachineBuilder = &stateMachineBuilder{}
var _ StateMachineConfigurator = &stateMachineBuilder{}

type StateMachineBuilder interface {
	OnNewToken(handler NewTokenHandler) StateMachineBuilder
	Build() *State
}

type StateMachineConfigurator interface {
	WithGrammar(grammar *Grammar) StateMachineBuilder
	WithStateMachineDefinition(definition *StateMachineDefinition) StateMachineBuilder
}

type stateMachineBuilder struct {
	definition *StateMachineDefinition
	onNewToken NewTokenHandler
}

func (smb *stateMachineBuilder) WithGrammar(grammar *Grammar) StateMachineBuilder {
	smb.definition = grammar.toStateMachineDefinition()
	return smb
}

func (smb *stateMachineBuilder) WithStateMachineDefinition(definition *StateMachineDefinition) StateMachineBuilder {
	smb.definition = definition
	return smb
}

func (smb *stateMachineBuilder) OnNewToken(handler NewTokenHandler) StateMachineBuilder {
	smb.onNewToken = handler
	return smb
}

func (smb *stateMachineBuilder) Build() *State {

	stateMap := make(map[string]*State)

	stateMap[StartState] = newStartState()
	stateMap[EndState] = newEndState()

	// build all the tokens
	for _, t := range smb.definition.States {
		stateMap[t.Name] = NewStateBuilder(t.Name).
			Family(t.Family).
			AcceptPattern(t.AcceptPattern).
			OnNewToken(smb.onNewToken).
			Build()
	}

	// add all the transitions
	for _, transition := range smb.definition.Transitions {
		currentState := stateMap[transition.StateName]
		for _, targetStateName := range transition.ValidTransitions {
			targetState := stateMap[targetStateName]
			currentState.addNextState(targetState)
		}
	}

	return stateMap[StartState]
}

func NewStateMachineBuilder() StateMachineConfigurator {
	return &stateMachineBuilder{}
}
