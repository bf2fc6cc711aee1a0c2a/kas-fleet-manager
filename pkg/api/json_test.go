package api

import (
	"database/sql/driver"
	"testing"

	. "github.com/onsi/gomega"
)

func Test_JSON_Scan(t *testing.T) {
	tests := []struct {
		name    string
		json    JSON
		value   interface{}
		wantErr bool
	}{
		{
			name:    "returns an error when value is not a byte array",
			json:    JSON{},
			value:   "",
			wantErr: true,
		},
		{
			name:    "returns an error when an incorrectly formatted json is passed",
			json:    JSON{},
			value:   []byte("{"),
			wantErr: true,
		},

		{
			name:    "should not return an error when an correctly formatted json is passed",
			json:    JSON{},
			value:   []byte("{}"),
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			err := tt.json.Scan(tt.value)
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_JSON_Object(t *testing.T) {
	tests := []struct {
		name    string
		json    JSON
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "returns a nil map when JSON is nil",
			json:    nil,
			wantErr: false,
			want:    nil,
		},
		{
			name:    "returns an error when value is not a proper json",
			json:    JSON{},
			wantErr: true,
			want:    map[string]interface{}{},
		},
		{
			name:    "returns an empty map when value an empty json object",
			json:    JSON([]byte("{}")),
			wantErr: false,
			want:    map[string]interface{}{},
		},
		{
			name:    "returns an map of objects from json object",
			json:    JSON([]byte("{\"a\":\"1\"}")),
			wantErr: false,
			want: map[string]interface{}{
				"a": "1",
			},
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			object, err := tt.json.Object()
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(object).To(Equal(tt.want))
		})
	}
}

func Test_JSON_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    JSON
		want    driver.Value
		wantErr bool
	}{
		{
			name:    "Return []byte('null') if json is nil",
			json:    JSON([]byte(nil)),
			wantErr: false,
			want:    []byte("null"),
		},
		{
			name:    "Return json if json is not nil",
			json:    JSON([]byte("{}")),
			wantErr: false,
			want:    []byte("{}"),
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			object, err := tt.json.MarshalJSON()
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(object).To(Equal(tt.want))
		})
	}
}

func Test_JSON_UnmarshalJSON(t *testing.T) {
	m := JSON([]byte("{\"a\":\"1\"}"))
	tests := []struct {
		name    string
		json    *JSON
		data    []byte
		wantErr bool
	}{
		{
			name:    "returns an error when value is not a byte array",
			json:    nil,
			wantErr: true,
		},
		{
			name:    "returns nil when a json is passed through.",
			json:    &m,
			wantErr: false,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			err := tt.json.UnmarshalJSON(tt.data)
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}

func Test_JSON_Value(t *testing.T) {
	tests := []struct {
		name    string
		json    JSON
		want    driver.Value
		wantErr bool
	}{

		{
			name:    "Return json if not nil",
			json:    JSON([]byte("{}")),
			wantErr: false,
			want:    []byte("{}"),
		},
		{
			name:    "Return nil if json is nil",
			json:    JSON([]byte("null")),
			wantErr: false,
			want:    nil,
		},
		{
			name:    "Return nil if json is nil",
			json:    JSON([]byte(nil)),
			wantErr: false,
			want:    nil,
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.json.Value()
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(got == nil).To(Equal(tt.want == nil))
			if got != nil {
				Expect(got).To(Equal(tt.want))
			}
		})
	}
}
