package secrets

import (
	"testing"

	"github.com/spyzhov/ajson"
	"k8s.io/apimachinery/pkg/util/sets"

	. "github.com/onsi/gomega"
)

const exampleSchema1 = `
{
  "properties": {
    "queueNameOrArn": {
      "title": "Queue Name",
      "description": "The SQS Queue name or ARN",
      "type": "string"
    },
    "accessKey": {
      "title": "Access Key",
      "description": "The access key obtained from AWS",
      "oneOf": [
        {
          "description": "The access key obtained from AWS",
          "type": "string",
          "format": "password"
        },
        {
          "description": "An opaque reference to the access key",
          "type": "object",
          "properties": {}
        }
      ]
    },
    "kafka.topic": {
      "title": "Topic names",
      "type": "string"
    },
    "kafka.secret": {
      "title": "Topic secret",
      "oneOf": [
        {
          "type": "string",
          "format": "password"
        },
        {
          "type": "object",
          "properties": {}
        }
      ]
    }
  }
}
`

func Test_getSecretPaths(t *testing.T) {
	type args struct {
		schemaText string
	}

	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name: "SQSConnectorSchemaText",
			args: args{
				schemaText: exampleSchema1,
			},
			want:    []string{`["accessKey"]`, `["kafka.secret"]`},
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getPathsToPasswordFields([]byte(tt.args.schemaText))
			if (err != nil) != tt.wantErr {
				t.Errorf("getPathsToPasswordFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			Expect(sets.NewString(got...)).To(Equal(sets.NewString(tt.want...)))
		})
	}
}

func Test_changePasswordFields(t *testing.T) {
	RegisterTestingT(t)
	type args struct {
		schemaText string
		doc        string
		f          func(node *ajson.Node) error
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		want    string
	}{
		{
			name: "replace with empty object",
			args: args{
				schemaText: exampleSchema1,
				doc: `{
					"queueNameOrArn": "test",
					"accessKey": "test",
					"kafka.topic": "test",
					"kafka.secret": "test"
				}`,
				f: func(node *ajson.Node) error {
					if node.Type() == ajson.String {
						return node.SetObject(map[string]*ajson.Node{})
					}
					return nil
				},
			},
			want: `{
				"queueNameOrArn": "test",
				"accessKey": {},
				"kafka.topic": "test",
				"kafka.secret": {}
			}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := modifySecrets([]byte(tt.args.schemaText), []byte(tt.args.doc), tt.args.f)
			if (err != nil) != tt.wantErr {
				t.Errorf("modifySecrets() error = %v, wantErr %v", err, tt.wantErr)
			}
			Expect(got).Should(MatchJSON(tt.want))
		})
	}
}
