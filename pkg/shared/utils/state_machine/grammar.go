package state_machine

type TokenDefinition StateDefinition

// Grammar - When parsing strings, it is easier to speak about a grammar and tokens than dealing with 'states'. These objects are just
// aliases of the StateMachineDefinition and TransitionDefinition objects
type Grammar struct {
	Tokens      []TokenDefinition
	Transitions []TokenTransitions
}

type TokenTransitions struct {
	TokenName        string
	ValidTransitions []string
}

func (g *Grammar) toStateMachineDefinition() *StateMachineDefinition {
	ret := StateMachineDefinition{
		States:      nil,
		Transitions: nil,
	}
	for _, token := range g.Tokens {
		ret.States = append(ret.States, StateDefinition(token))
	}
	for _, transition := range g.Transitions {
		ret.Transitions = append(ret.Transitions, *transition.toStateTransitions())
	}

	return &ret
}

func (tt *TokenTransitions) toStateTransitions() *TransitionDefinition {
	return &TransitionDefinition{
		StateName:        tt.TokenName,
		ValidTransitions: tt.ValidTransitions,
	}
}
