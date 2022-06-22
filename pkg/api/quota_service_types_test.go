package api

import (
	"testing"

	. "github.com/onsi/gomega"
)

func Test_QuotaService(t *testing.T) {
	tests := []struct {
		name      string
		want      string
		wantErr   bool
		quotatype QuotaType
	}{
		{
			name:      "Transforms ams quota type to string",
			quotatype: AMSQuotaType,
			want:      "ams",
		},
		{
			name:      "Transforms list-management quota type to string",
			quotatype: QuotaManagementListQuotaType,
			want:      "quota-management-list",
		},

		{
			name:      "Transforms undefined quota type to string",
			quotatype: UndefinedQuotaType,
			want:      "",
		},
	}

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase

		t.Run(tt.name, func(t *testing.T) {
			res := tt.quotatype.String()
			Expect(res).To(Equal(tt.want))
		})
	}
}
