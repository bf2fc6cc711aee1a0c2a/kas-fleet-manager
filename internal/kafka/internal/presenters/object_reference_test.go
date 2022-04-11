package presenters

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/compat"
	mockKafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	mockObjRef "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/object_reference"
	mockServAcc "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/service_accounts"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"

	. "github.com/onsi/gomega"
)

func TestPresentReference(t *testing.T) {
	type args struct {
		id, obj interface{}
	}

	e := errors.GeneralError("test")

	tests := []struct {
		name string
		args args
		want compat.ObjectReference
	}{
		{
			name: "should return valid ObjectReference for kafka",
			args: args{
				id:  mockObjRef.GetObjectReferenceMockId(""),
				obj: mockKafka.BuildKafkaRequest(nil),
			},
			want: mockObjRef.GetKafkaObjectReference(),
		},
		{
			name: "should return valid ObjectReference for error",
			args: args{
				id:  mockObjRef.GetObjectReferenceMockId(""),
				obj: e,
			},
			want: mockObjRef.GetErrorObjectReference(),
		},
		{
			name: "should return valid ObjectReference for service account",
			args: args{
				id:  mockObjRef.GetObjectReferenceMockId(""),
				obj: mockServAcc.BuildApiServiceAccount(nil),
			},
			want: mockObjRef.GetServiceAccountObjectReference(),
		},
		{
			name: "should return valid ObjectReference for non referenced object",
			args: args{
				id:  mockObjRef.GetObjectReferenceMockId(""),
				obj: mockObjRef.GetObjectReferenceMockId(""),
			},
			want: mockObjRef.GetEmptyObjectReference(),
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Expect(PresentReference(tt.args.id, tt.args.obj)).To(Equal(tt.want))
		})
	}
}
