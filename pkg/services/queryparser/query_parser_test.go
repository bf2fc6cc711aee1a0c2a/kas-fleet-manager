package services

import (
	. "github.com/onsi/gomega"
	"testing"
)

func TestABC_QueryParser(t *testing.T) {

	tests := []struct {
		name    string
		qry     string
		wantErr bool
	}{
		{
			name:    "Testing just `=` sign",
			qry:     "=",
			wantErr: true,
		},
		{
			name:    "Testing incomplete query",
			qry:     "name=",
			wantErr: true,
		},
		{
			name:    "Testing incomplete join",
			qry:     "name='test' and ",
			wantErr: true,
		},
		{
			name:    "Testing escaped quote",
			qry:     `name='test\'123'`,
			wantErr: false,
		},
		{
			name:    "Testing wrong unescaped quote",
			qry:     `name='test'123'`,
			wantErr: true,
		},
		{
			name:    "Complex query with braces",
			qry:     "((cloud_provider = value and name = value1) and (owner = value2 or region=b ) ) or owner=c or name=e and region LIKE '%test%'",
			wantErr: false,
		},
		{
			name: "10 JOINS (maximum allowed)",
			qry: "name = value1 " +
				"and name = value2 " +
				"and name = value3 " +
				"or name = value4 " +
				"and name=value5 " +
				"and name = value6 " +
				"and name = value7 " +
				"and name = value8 " +
				"and name = value9 " +
				"and name = value10 " +
				"or name = value11",
			wantErr: false,
		},
		{
			name: "11 JOINS (too many)",
			qry: "name = value1 " +
				"and name = value2 " +
				"and name = value3 " +
				"or name = value4 " +
				"and name=value5 " +
				"and name = value6 " +
				"and name = value7 " +
				"and name = value8 " +
				"and name = value9 " +
				"and name = value10 " +
				"or name = value11 " +
				"and name = value12",
			wantErr: true,
		},
		{
			name:    "Complex query with unbalanced braces",
			qry:     "((cloud_provider = value and name = value1) and (owner = value2 or region=b  ) or owner=c or name=e and region LIKE '%test%'",
			wantErr: true,
		},
		{
			name:    "Bad column name",
			qry:     "badcolumn=test",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterTestingT(t)
			_, err := NewQueryParser().Parse(tt.qry)

			if err != nil && !tt.wantErr {
				t.Errorf("QueryParser() error = %v, wantErr = %v", err, tt.wantErr)
			}
			Expect(err != nil).To(Equal(tt.wantErr))
		})
	}
}
