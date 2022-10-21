package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
	errors "github.com/pkg/errors"
)

type Route53API route53iface.Route53API

const (
	DefaultAWSRoute53Region = "us-east-1"
	DefaultGCPRoute53Region = "us-east-1"
)

//go:generate moq -out client_moq.go . AWSClient
type AWSClient interface {
	// route53
	ListHostedZonesByNameInput(dnsName string) (*route53.ListHostedZonesByNameOutput, error)
	ChangeResourceRecordSets(dnsName string, recordChangeBatch *route53.ChangeBatch) (*route53.ChangeResourceRecordSetsOutput, error)
	GetChange(changeId string) (*route53.GetChangeOutput, error)
}

type ClientFactory interface {
	NewClient(credentials Config, region string) (AWSClient, error)
}

type DefaultClientFactory struct{}

func (f *DefaultClientFactory) NewClient(credentials Config, region string) (AWSClient, error) {
	return newClient(credentials, region)
}

func NewDefaultClientFactory() *DefaultClientFactory {
	return &DefaultClientFactory{}
}

type MockClientFactory struct {
	mock AWSClient
}

func (m *MockClientFactory) NewClient(credentials Config, region string) (AWSClient, error) {
	return m.mock, nil
}

func NewMockClientFactory(client AWSClient) *MockClientFactory {
	return &MockClientFactory{
		mock: client,
	}
}

var _ AWSClient = &awsCl{}

type awsCl struct {
	route53Client route53iface.Route53API
}

// Config contains the AWS settings
type Config struct {
	// AccessKeyID is the AWS access key identifier.
	AccessKeyID string
	// SecretAccessKey is the AWS secret access key.
	SecretAccessKey string
}

func newClient(credentials Config, region string) (AWSClient, error) {
	cfg := &aws.Config{
		Credentials: awscredentials.NewStaticCredentials(
			credentials.AccessKeyID,
			credentials.SecretAccessKey,
			""),
		Region:  aws.String(region),
		Retryer: client.DefaultRetryer{NumMaxRetries: 2},
	}
	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, err
	}
	return &awsCl{
		route53Client: route53.New(sess),
	}, nil
}

func (client *awsCl) GetChange(changeId string) (*route53.GetChangeOutput, error) {
	changeInput := &route53.GetChangeInput{
		Id: &changeId,
	}

	change, err := client.route53Client.GetChange(changeInput)
	if err != nil {
		return nil, wrapAWSError(err, "Failed to get Change.")
	}

	return change, nil
}

func (client *awsCl) ListHostedZonesByNameInput(dnsName string) (*route53.ListHostedZonesByNameOutput, error) {
	maxItems := "1"
	requestInput := &route53.ListHostedZonesByNameInput{
		DNSName:  &dnsName,
		MaxItems: &maxItems,
	}

	zone, err := client.route53Client.ListHostedZonesByName(requestInput)
	if err != nil {
		return nil, wrapAWSError(err, "Failed to get DNS zone.")
	}
	return zone, nil
}

func (client *awsCl) ChangeResourceRecordSets(dnsName string, recordChangeBatch *route53.ChangeBatch) (*route53.ChangeResourceRecordSetsOutput, error) {
	zones, err := client.ListHostedZonesByNameInput(dnsName)
	if err != nil {
		return nil, err
	}
	if len(zones.HostedZones) == 0 {
		return nil, fmt.Errorf("No Hosted Zones found")
	}

	hostedZoneId := zones.HostedZones[0].Id

	recordChanges := &route53.ChangeResourceRecordSetsInput{
		HostedZoneId: hostedZoneId,
		ChangeBatch:  recordChangeBatch,
	}

	recordSetsOutput, err := client.route53Client.ChangeResourceRecordSets(recordChanges)

	if err != nil {
		awsErr := err.(awserr.Error)
		if awsErr.Code() == "InvalidChangeBatch" {
			errorMessage := awsErr.Message()

			// Record set not created in the first place
			recordSetNotFound := strings.Contains(errorMessage, "but it was not found")

			// Kafka cluster failed to create on the cluster, we have an entry in the database.
			recordSetDomainNameEmpty := strings.Contains(errorMessage, "Domain name is empty")

			// Record set has already been created
			recordSetAlreadyExists := strings.Contains(errorMessage, "but it already exists")

			if recordSetNotFound || recordSetDomainNameEmpty || recordSetAlreadyExists {
				return nil, nil
			}
		}
		return nil, wrapAWSError(err, "Failed to get DNS zone.")
	}
	return recordSetsOutput, nil
}

func wrapAWSError(err error, msg string) error {
	switch err.(type) {
	case awserr.RequestFailure:
		return errors.Wrapf(err, msg)
	default:
		return err
	}
}
