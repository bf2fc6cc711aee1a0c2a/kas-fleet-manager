package handlers

import (
	"encoding/json"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/onsi/gomega"
	"testing"
)

func Test_ValidateErrorHandler(t *testing.T) {
	type assertion func(eh public.ErrorHandler, g *gomega.WithT)

	type test struct {
		name         string
		errorHandler public.ErrorHandler
		assertion
	}

	tests := []test{
		{
			name:         "No explicit ErrorHandler",
			errorHandler: public.ErrorHandler{},
			assertion: func(eh public.ErrorHandler, g *gomega.WithT) {
				g.Expect(validateProcessorErrorHandler(&eh)).To(gomega.BeNil())
				g.Expect(eh.Stop).ToNot(gomega.BeNil())
				g.Expect(eh.Log).To(gomega.BeNil())
				g.Expect(eh.DeadLetterQueue).To(gomega.Equal(public.ErrorHandlerDeadLetterQueueDeadLetterQueue{}))
			},
		},
		{
			name:         "Explicit ErrorHandler::Stop",
			errorHandler: public.ErrorHandler{Stop: map[string]interface{}{}},
			assertion: func(eh public.ErrorHandler, g *gomega.WithT) {
				g.Expect(validateProcessorErrorHandler(&eh)).To(gomega.BeNil())
				g.Expect(eh.Stop).ToNot(gomega.BeNil())
				g.Expect(eh.Log).To(gomega.BeNil())
				g.Expect(eh.DeadLetterQueue).To(gomega.Equal(public.ErrorHandlerDeadLetterQueueDeadLetterQueue{}))
			},
		},
		{
			name:         "Explicit ErrorHandler::Log",
			errorHandler: public.ErrorHandler{Log: map[string]interface{}{}},
			assertion: func(eh public.ErrorHandler, g *gomega.WithT) {
				g.Expect(validateProcessorErrorHandler(&eh)).To(gomega.BeNil())
				g.Expect(eh.Stop).To(gomega.BeNil())
				g.Expect(eh.Log).ToNot(gomega.BeNil())
				g.Expect(eh.DeadLetterQueue).To(gomega.Equal(public.ErrorHandlerDeadLetterQueueDeadLetterQueue{}))
			},
		},
		{
			name:         "Explicit ErrorHandler::DeadLetterQueue",
			errorHandler: public.ErrorHandler{DeadLetterQueue: public.ErrorHandlerDeadLetterQueueDeadLetterQueue{Topic: "dlq"}},
			assertion: func(eh public.ErrorHandler, g *gomega.WithT) {
				g.Expect(validateProcessorErrorHandler(&eh)).To(gomega.BeNil())
				g.Expect(eh.Stop).To(gomega.BeNil())
				g.Expect(eh.Log).To(gomega.BeNil())
				g.Expect(eh.DeadLetterQueue).ToNot(gomega.Equal(public.ErrorHandlerDeadLetterQueueDeadLetterQueue{}))
			},
		},
		{
			name:         "Explicit ErrorHandlers::Multiple defined",
			errorHandler: public.ErrorHandler{Log: map[string]interface{}{}, Stop: map[string]interface{}{}},
			assertion: func(eh public.ErrorHandler, g *gomega.WithT) {
				g.Expect(validateProcessorErrorHandler(&eh)).ToNot(gomega.BeNil())
			},
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			tt.assertion(tt.errorHandler, g)
		})
	}
}

func Test_ValidateProcessorRequest(t *testing.T) {
	type assertion func(request public.ProcessorRequest, g *gomega.WithT)

	type test struct {
		name    string
		request public.ProcessorRequest
		assertion
	}

	valid := `
			{
				"from": {
					"uri": "kafka:my-topic",
					"steps": [
						{"unmarshal": { "json":{} } },
						{"log": "this is my processor"},
						{"to": {
							"uri":"kafka",
							"parameters": { "topic":"my-output-topic" }
							}
						}
					]
				}
			}`

	invalid := `{ "bananna":{} }`

	var validJson map[string]interface{}
	var invalidJson map[string]interface{}
	if err := json.Unmarshal([]byte(valid), &validJson); err != nil {
		panic("Unable to convert valid Json string to Json object")
	}
	if err := json.Unmarshal([]byte(invalid), &invalidJson); err != nil {
		panic("Unable to convert invalid Json string to Json object")
	}

	// Setup schema for Processor validation
	processorType, err := SetupValidProcessorSchema()
	if err != nil {
		panic("Unable to convert Json schema string to Json object")
	}
	processorTypesService := &services.ProcessorTypesServiceMock{
		GetFunc: func(processorTypeId string) (*dbapi.ProcessorType, *errors.ServiceError) {
			return processorType, nil
		},
	}

	tests := []test{
		{
			name:    "Valid",
			request: public.ProcessorRequest{Definition: validJson, Channel: "stable"},
			assertion: func(request public.ProcessorRequest, g *gomega.WithT) {
				g.Expect(validateProcessorRequest(processorTypesService, &request)()).To(gomega.BeNil())
			},
		},
		{
			name:    "Invalid::1",
			request: public.ProcessorRequest{Definition: invalidJson, Channel: "stable"},
			assertion: func(request public.ProcessorRequest, g *gomega.WithT) {
				g.Expect(validateProcessorRequest(processorTypesService, &request)()).ToNot(gomega.BeNil())
			},
		},
		{
			name:    "Invalid::Empty",
			request: public.ProcessorRequest{},
			assertion: func(request public.ProcessorRequest, g *gomega.WithT) {
				g.Expect(validateProcessorRequest(processorTypesService, &request)()).ToNot(gomega.BeNil())
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			tt.assertion(tt.request, g)
		})
	}
}

func Test_ValidateProcessor(t *testing.T) {
	type assertion func(processor public.Processor, g *gomega.WithT)

	type test struct {
		name      string
		processor public.Processor
		assertion
	}

	valid := `
			{
				"from": {
					"uri": "kafka:my-topic",
					"steps": [
						{"unmarshal": { "json":{} } },
						{"log": "this is my processor"},
						{"to": {
							"uri":"kafka",
							"parameters": { "topic":"my-output-topic" }
							}
						}
					]
				}
			}`

	invalid := `{ "bananna":{} }`

	var validJson map[string]interface{}
	var invalidJson map[string]interface{}
	if err := json.Unmarshal([]byte(valid), &validJson); err != nil {
		panic("Unable to convert valid Json string to Json object")
	}
	if err := json.Unmarshal([]byte(invalid), &invalidJson); err != nil {
		panic("Unable to convert invalid Json string to Json object")
	}

	// Setup schema for Processor validation
	processorType, err := SetupValidProcessorSchema()
	if err != nil {
		panic("Unable to convert Json schema string to Json object")
	}
	processorTypesService := &services.ProcessorTypesServiceMock{
		GetFunc: func(processorTypeId string) (*dbapi.ProcessorType, *errors.ServiceError) {
			return processorType, nil
		},
	}

	tests := []test{
		{
			name:      "Valid",
			processor: public.Processor{Definition: validJson, Channel: "stable"},
			assertion: func(processor public.Processor, g *gomega.WithT) {
				g.Expect(validateProcessor(processorTypesService, &processor)()).To(gomega.BeNil())
			},
		},
		{
			name:      "Invalid::1",
			processor: public.Processor{Definition: invalidJson, Channel: "stable"},
			assertion: func(processor public.Processor, g *gomega.WithT) {
				g.Expect(validateProcessor(processorTypesService, &processor)()).ToNot(gomega.BeNil())
			},
		},
		{
			name:      "Invalid::Empty",
			processor: public.Processor{},
			assertion: func(processor public.Processor, g *gomega.WithT) {
				g.Expect(validateProcessor(processorTypesService, &processor)()).ToNot(gomega.BeNil())
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			tt.assertion(tt.processor, g)
		})
	}
}
