package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	"github.com/onsi/gomega"
)

var (
	testValue  = "test"
	testConfig = Config{
		AccessKeyID:     testValue,
		SecretAccessKey: testValue,
	}

	isTruncated = false
	intString   = "2"

	testHostedZones = &route53.ListHostedZonesByNameOutput{
		DNSName:     &testValue,
		IsTruncated: &isTruncated,
		MaxItems:    &intString,
		HostedZones: []*route53.HostedZone{
			{
				CallerReference: &intString,
				Id:              &intString,
				Name:            &testValue,
			},
		},
	}
	awsErr = awserr.New("InvalidChangeBatch", "but it already exists", nil)
)

type testClientFactory struct{}

func (t testClientFactory) NewClient(route53Client *route53iface.Route53API) AWSClient {
	return &awsCl{
		route53Client: *route53Client,
	}
}

func TestAwsClient_NewClientFromConfig(t *testing.T) {
	type args struct {
		credentials Config
		region      string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Should return new Client from config",
			args: args{
				credentials: testConfig,
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			_, err := newClient(tt.args.credentials, tt.args.region)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestAwsClient_NewClientFromFactory(t *testing.T) {
	type args struct {
		credentials Config
		region      string
	}
	type fields struct {
		f *DefaultClientFactory
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "Should return new Client from factory",
			args: args{
				credentials: testConfig,
			},
			fields: fields{
				f: NewDefaultClientFactory(),
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			_, err := tt.fields.f.NewClient(tt.args.credentials, tt.args.region)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestAwsClient_GetChange(t *testing.T) {
	type fields struct {
		route53Client route53iface.Route53API
	}
	type args struct {
		changeId string
	}
	tests := []struct {
		name    string
		args    args
		fields  fields
		wantErr bool
	}{
		{
			name: "Should return an error if GetChange from route53Client fails",
			args: args{
				changeId: testValue,
			},
			fields: fields{
				route53Client: &Route53APIMock{
					GetChangeFunc: func(in1 *route53.GetChangeInput) (*route53.GetChangeOutput, error) {
						return nil, awsErr
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should successfully execute ChangeOutput",
			args: args{
				changeId: testValue,
			},
			fields: fields{
				route53Client: &Route53APIMock{
					GetChangeFunc: func(in1 *route53.GetChangeInput) (*route53.GetChangeOutput, error) {
						return &route53.GetChangeOutput{}, nil
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			awsClient := testClientFactory{}.NewClient(&tt.fields.route53Client)
			_, err := awsClient.GetChange(testValue)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestAwsClient_ListHostedZonesByNameInput(t *testing.T) {
	type fields struct {
		route53Client route53iface.Route53API
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Should fail to return list of hosted zones by name output",
			fields: fields{
				route53Client: &Route53APIMock{
					ListHostedZonesByNameFunc: func(in1 *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error) {
						return nil, awsErr
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should successfully return list of hosted zones by name output",
			fields: fields{
				route53Client: &Route53APIMock{
					ListHostedZonesByNameFunc: func(in1 *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error) {
						return &route53.ListHostedZonesByNameOutput{}, nil
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			awsClient := testClientFactory{}.NewClient(&tt.fields.route53Client)
			_, err := awsClient.ListHostedZonesByNameInput(testValue)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}

func TestAwsClient_ChangeResourceRecordSets(t *testing.T) {
	type fields struct {
		route53Client route53iface.Route53API
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Should fail when ListHostedZonesByNameInput returns an error",
			fields: fields{
				route53Client: &Route53APIMock{
					ListHostedZonesByNameFunc: func(in1 *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error) {
						return nil, awsErr
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should fail when ListHostedZonesByNameInput returns an empty list of hosted zones",
			fields: fields{
				route53Client: &Route53APIMock{
					ListHostedZonesByNameFunc: func(in1 *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error) {
						return &route53.ListHostedZonesByNameOutput{}, nil
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should fail when ChangeResourceRecordSets returns an error",
			fields: fields{
				route53Client: &Route53APIMock{
					ListHostedZonesByNameFunc: func(in1 *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error) {
						return testHostedZones, nil
					},
					ChangeResourceRecordSetsFunc: func(in1 *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
						return nil, awserr.New("InvalidChangeBatch", "test", nil)
					},
				},
			},
			wantErr: true,
		},
		{
			name: "Should not fail when ChangeResourceRecordSets returns an error with expected message",
			fields: fields{
				route53Client: &Route53APIMock{
					ListHostedZonesByNameFunc: func(in1 *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error) {
						return testHostedZones, nil
					},
					ChangeResourceRecordSetsFunc: func(in1 *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
						return nil, awserr.New("InvalidChangeBatch", "but it already exists", nil)
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Should successfully execute ChangeResourceRecordSets",
			fields: fields{
				route53Client: &Route53APIMock{
					ListHostedZonesByNameFunc: func(in1 *route53.ListHostedZonesByNameInput) (*route53.ListHostedZonesByNameOutput, error) {
						return testHostedZones, nil
					},
					ChangeResourceRecordSetsFunc: func(in1 *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
						return &route53.ChangeResourceRecordSetsOutput{}, nil
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			awsClient := testClientFactory{}.NewClient(&tt.fields.route53Client)
			_, err := awsClient.ChangeResourceRecordSets(testValue, &route53.ChangeBatch{})
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
		})
	}
}
