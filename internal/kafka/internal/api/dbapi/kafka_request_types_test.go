package dbapi

import (
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/config"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/onsi/gomega"
)

func TestKafkaRequest_DesiredBillingModelIsEnterprise(t *testing.T) {
	type fields struct {
		DesiredKafkaBillingModel string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "return true if enterprise is the desired billing model",
			fields: fields{
				DesiredKafkaBillingModel: constants.BillingModelEnterprise.String(),
			},
			want: true,
		},
		{
			name: "return false if enterprise is not the desired billing model",
			fields: fields{
				DesiredKafkaBillingModel: "something else",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			k := &KafkaRequest{
				DesiredKafkaBillingModel: testcase.fields.DesiredKafkaBillingModel,
			}
			got := k.DesiredBillingModelIsEnterprise()
			g.Expect(got).To(gomega.Equal(testcase.want))
		})
	}
}

func TestKafkaRequest_HasCertificateInfo(t *testing.T) {
	tests := []struct {
		name         string
		kafkaRequest *KafkaRequest
		want         bool
	}{
		{
			name: "return false if cert info are not set for the given kafka",
			kafkaRequest: &KafkaRequest{
				KafkasRoutesBaseDomainName:      "",
				KafkasRoutesBaseDomainTLSKeyRef: "",
				KafkasRoutesBaseDomainTLSCrtRef: "",
			},
			want: false,
		},
		{
			name: "return true if cert info are set for the given kafka",
			kafkaRequest: &KafkaRequest{
				KafkasRoutesBaseDomainName:      "domain",
				KafkasRoutesBaseDomainTLSKeyRef: "crt-ref",
				KafkasRoutesBaseDomainTLSCrtRef: "key-ref",
			},
			want: true,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			got := testcase.kafkaRequest.HasCertificateInfo()
			g.Expect(got).To(gomega.Equal(testcase.want))
		})
	}
}

func TestKafkaRequest_IsUsingSharedTLSCertificate(t *testing.T) {
	type fields struct {
		KafkasRoutesBaseDomainName string
	}
	type args struct {
		kafkaConfig *config.KafkaConfig
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "return true if the kafkas routes base domain is same as the kafka domain name given in the configuration",
			fields: fields{
				KafkasRoutesBaseDomainName: "kafka.bf2.dev",
			},
			args: args{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "kafka.bf2.dev",
				},
			},
			want: true,
		},
		{
			name: "return true if the kafkas routes base domain is prefixed with 'trial'",
			fields: fields{
				KafkasRoutesBaseDomainName: "trial.kafka.bf2.dev",
			},
			args: args{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "kafka.bf2.dev",
				},
			},
			want: true,
		},
		{
			name: "return false if the kafkas routes base domain is different from the kafka domain name given in the configuration",
			fields: fields{
				KafkasRoutesBaseDomainName: "1234.kafka.bf2.dev",
			},
			args: args{
				kafkaConfig: &config.KafkaConfig{
					KafkaDomainName: "kafka.bf2.dev",
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			k := &KafkaRequest{
				KafkasRoutesBaseDomainName: testcase.fields.KafkasRoutesBaseDomainName,
			}
			got := k.IsUsingSharedTLSCertificate(testcase.args.kafkaConfig)
			g.Expect(got).To(gomega.Equal(testcase.want))
		})
	}
}

func TestKafkaRequest_IsADeveloperInstance(t *testing.T) {
	type fields struct {
		InstanceType string
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "return true if the instance type is developer",
			fields: fields{
				InstanceType: types.DEVELOPER.String(),
			},
			want: true,
		},
		{
			name: "return false if the instance type is not developer",
			fields: fields{
				InstanceType: types.STANDARD.String(),
			},
			want: false,
		},
	}
	for _, tt := range tests {
		testcase := tt
		t.Run(testcase.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			t.Parallel()
			k := &KafkaRequest{
				InstanceType: testcase.fields.InstanceType,
			}
			got := k.IsADeveloperInstance()
			g.Expect(got).To(gomega.Equal(testcase.want))
		})
	}
}
