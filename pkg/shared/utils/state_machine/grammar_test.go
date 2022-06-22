package state_machine

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_TokenTransitions_toStateTransitions(t *testing.T) {
	tests := []struct {
		name        string
		transitions *TokenTransitions
		want        *TransitionDefinition
	}{
		{
			name: "return TransitionDefinition",
			transitions: &TokenTransitions{
				TokenName:        "testToken",
				ValidTransitions: []string{"DIVIDEND"},
			},
			want: &TransitionDefinition{
				StateName:        "testToken",
				ValidTransitions: []string{"DIVIDEND"},
			},
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.transitions.toStateTransitions()).To(Equal(tt.want))
		})
	}
}

func Test_Grammar_toStateMachineDefinition(t *testing.T) {
	type fields struct {
		Tokens      []TokenDefinition
		Transitions []TokenTransitions
	}
	tests := []struct {
		name    string
		fields  fields
		grammar *Grammar
		want    *StateMachineDefinition
	}{
		{
			name: "return StateMachineDefinition",
			grammar: &Grammar{
				Tokens: []TokenDefinition{
					{
						Name:          "NEW",
						AcceptPattern: "NEW",
					},
				},
				Transitions: []TokenTransitions{
					{
						TokenName:        "NEW",
						ValidTransitions: []string{"NEW"},
					},
				},
			},
			want: &StateMachineDefinition{
				States: []StateDefinition{
					{
						Name:          "NEW",
						Family:        "",
						AcceptPattern: "NEW",
						OnNewToken:    nil,
					},
				},
				Transitions: []TransitionDefinition{
					{
						StateName:        "NEW",
						ValidTransitions: []string{"NEW"},
					},
				},
			},
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			Expect(tt.grammar.toStateMachineDefinition()).To(Equal(tt.want))
		})
	}
}
