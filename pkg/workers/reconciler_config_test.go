package workers

import (
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/spf13/pflag"
)

func Test_AddFlags(t *testing.T) {
	emptyFlagSet := &pflag.FlagSet{}
	config := NewReconcilerConfig()
	config.AddFlags(emptyFlagSet)

	g := gomega.NewWithT(t)

	flag, err := emptyFlagSet.GetDuration("reconciler-repeat-interval")

	g.Expect(err).NotTo(gomega.HaveOccurred())
	g.Expect(flag).To(gomega.Equal(30 * time.Second))
}

func TestReconcilerConfig_ReadFiles(t *testing.T) {
	type fields struct {
		ReconcilerRepeatInterval               time.Duration
		LeaderLeaseExpirationTime              time.Duration
		LeaderElectionReconcilerRepeatInterval time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "should always return nil",
			fields: fields{
				ReconcilerRepeatInterval:               30 * time.Second,
				LeaderLeaseExpirationTime:              1 * time.Minute,
				LeaderElectionReconcilerRepeatInterval: 15 * time.Second,
			},
			wantErr: false,
		},
	}
	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			c := &ReconcilerConfig{
				ReconcilerRepeatInterval:               tt.fields.ReconcilerRepeatInterval,
				LeaderLeaseExpirationTime:              tt.fields.LeaderLeaseExpirationTime,
				LeaderElectionReconcilerRepeatInterval: tt.fields.LeaderElectionReconcilerRepeatInterval,
			}
			g.Expect(c.ReadFiles() != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
