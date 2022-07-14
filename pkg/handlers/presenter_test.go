package handlers

import (
	"fmt"
	"testing"

	"github.com/onsi/gomega"
)

var (
	kind = "test_kind"
	id   = "test-id"
	obj  = "test-obj"
)

func testObjectPath(id string, obj interface{}) string {
	return fmt.Sprintf("test/%s", id)
}

func testObjectKind(i interface{}) string {
	return kind
}

func Test_PresentReferenceWith(t *testing.T) {
	type args struct {
		id         interface{}
		obj        interface{}
		objectKind func(i interface{}) string
		objectPath func(id string, obj interface{}) string
	}

	tests := []struct {
		name     string
		args     args
		wantId   string
		wantKind string
		wantHref string
	}{
		{
			name: "Should present Object reference",
			args: args{
				id:         id,
				obj:        obj,
				objectKind: testObjectKind,
				objectPath: testObjectPath,
			},
			wantId:   id,
			wantKind: kind,
			wantHref: "test/test-id",
		},
		{
			name: "Should present Object reference with id passed as a pointer",
			args: args{
				id:         &id,
				obj:        obj,
				objectKind: testObjectKind,
				objectPath: testObjectPath,
			},
			wantId:   id,
			wantKind: kind,
			wantHref: "test/test-id",
		},
		{
			name: "Should present Object reference with empty values",
			args: args{
				id:         "",
				obj:        "",
				objectKind: testObjectKind,
				objectPath: testObjectPath,
			},
			wantId:   "",
			wantKind: "",
			wantHref: "",
		},
	}

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			ref := PresentReferenceWith(tt.args.id, tt.args.obj, tt.args.objectKind, tt.args.objectPath)
			g.Expect(ref.Id).To(gomega.Equal(tt.wantId))
			g.Expect(ref.Kind).To(gomega.Equal(tt.wantKind))
			g.Expect(ref.Href).To(gomega.Equal(tt.wantHref))
		})
	}
}
