package state_machine

import (
	. "github.com/onsi/gomega"
	"testing"
)

// createStateMachine - creates a simple StateMachine that validates transitions for an issue state
func createStateMachine() *State {
	definition := StateMachineDefinition{
		States: []StateDefinition{
			{Name: "NEW", AcceptPattern: "NEW"},
			{Name: "ASSIGNED", AcceptPattern: "ASSIGNED"},
			{Name: "IN PROGRESS", AcceptPattern: "IN PROGRESS"},
			{Name: "WAITING FOR REVIEW", AcceptPattern: "WAITING FOR REVIEW"},
			{Name: "REVIEWING", AcceptPattern: "REVIEWING"},
			{Name: "WAITING FOR RELEASE", AcceptPattern: "WAITING FOR RELEASE"},
			{Name: "WON'T DO", AcceptPattern: "WON'T DO"},
			{Name: "DONE", AcceptPattern: "DONE"},
		},
		Transitions: []TransitionDefinition{
			{StateName: StartState, ValidTransitions: []string{"NEW"}},
			{StateName: "NEW", ValidTransitions: []string{"ASSIGNED", "WON'T DO"}},
			{StateName: "ASSIGNED", ValidTransitions: []string{"IN PROGRESS", "WON'T DO"}},
			{StateName: "IN PROGRESS", ValidTransitions: []string{"WAITING FOR REVIEW", "ASSIGNED", "WON'T DO"}},
			{StateName: "WAITING FOR REVIEW", ValidTransitions: []string{"IN PROGRESS", "REVIEWING", "WON'T DO"}},
			{StateName: "REVIEWING", ValidTransitions: []string{"IN PROGRESS", "WON'T DO", "WAITING FOR RELEASE"}},
			{StateName: "WAITING FOR RELEASE", ValidTransitions: []string{"DONE", "WON'T DO", "IN PROGRESS"}},
			{StateName: "DONE", ValidTransitions: []string{EndState}},
			{StateName: "WON'T DO", ValidTransitions: []string{EndState}},
		},
	}
	return NewStateMachineBuilder().
		WithStateMachineDefinition(&definition).
		Build()
}

func Test_StateMachine_ValidPaths(t *testing.T) {
	tests := []struct {
		name string
		path []string
	}{
		{
			name: "Valid path - WON'T DO",
			path: []string{"NEW", "ASSIGNED", "WON'T DO"},
		},
		{
			name: "Valid path - AFTER REVIEWING: WON'T DO",
			path: []string{"NEW", "ASSIGNED", "IN PROGRESS", "WAITING FOR REVIEW", "REVIEWING", "WON'T DO"},
		},
		{
			name: "Valid path - DONE",
			path: []string{"NEW", "ASSIGNED", "IN PROGRESS", "WAITING FOR REVIEW", "REVIEWING", "WAITING FOR RELEASE", "DONE"},
		},
		{
			name: "Valid path - DONE With Reiteration",
			path: []string{"NEW", "ASSIGNED", "IN PROGRESS", "WAITING FOR REVIEW", "IN PROGRESS", "WAITING FOR REVIEW", "REVIEWING", "WAITING FOR RELEASE", "DONE"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			initialState := createStateMachine()
			currentState := initialState
			var err error
			for _, s := range tt.path {
				currentState, err = currentState.Move(s)
				Expect(err).ToNot(HaveOccurred())
			}
			Expect(currentState.Eof()).To(BeTrue())
		})
	}
}

func Test_StateMachine_InvalidPaths(t *testing.T) {
	tests := []struct {
		name  string
		path  []string
		error string
	}{
		{
			name:  "Invalid path - DONE",
			path:  []string{"NEW", "ASSIGNED", "IN PROGRESS", "DONE"},
			error: "Unexpected Token `DONE`",
		},
		{
			name:  "Invalid path - ASSIGNED",
			path:  []string{"NEW", "ASSIGNED", "IN PROGRESS", "WAITING FOR REVIEW", "ASSIGNED"},
			error: "Unexpected Token `ASSIGNED`",
		},
		{
			name:  "Invalid path - END STATE NOT REACHED",
			path:  []string{"NEW", "ASSIGNED", "IN PROGRESS", "WAITING FOR REVIEW"},
			error: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			initialState := createStateMachine()
			currentState := initialState
			var err error
			for _, s := range tt.path {
				currentState, err = currentState.Move(s)
				if err != nil {
					Expect(err.Error()).To(Equal(tt.error))
				}
			}
			if currentState != nil {
				Expect(currentState.Eof()).To(BeFalse()) // in this test we never end successfully
			}
		})
	}
}
