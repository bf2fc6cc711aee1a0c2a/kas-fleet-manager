package kafka_tls_certificate_management

import (
	"testing"

	"github.com/onsi/gomega"
)

func TestParseReason(t *testing.T) {
	type args struct {
		reason int
	}
	tests := []struct {
		name    string
		args    args
		want    CertificateRevocationReason
		wantErr bool
	}{
		{
			name: "return an error when reason is not valid as specified by https://www.rfc-editor.org/rfc/rfc5280#section-5.3.1",
			args: args{
				reason: 19090,
			},
			want:    CertificateRevocationReason(-1),
			wantErr: true,
		},
		{
			name: "return the certificate revocation reason",
			args: args{
				reason: 0,
			},
			want:    Unspecified,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			got, err := ParseReason(testcase.args.reason)
			g.Expect(got).To(gomega.Equal(testcase.want))
			g.Expect(err != nil).To(gomega.Equal(testcase.wantErr))
		})
	}
}
