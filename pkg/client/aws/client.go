package aws

import (
	"fmt"
	"strings"

	errors "github.com/zgalor/weberr"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	awscredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
)

type Client interface {
	// route53
	ListHostedZonesByNameInput(dnsName string) (*route53.ListHostedZonesByNameOutput, error)
	ChangeResourceRecordSets(dnsName string, recordChangeBatch *route53.ChangeBatch) (*route53.ChangeResourceRecordSetsOutput, error)
}

type awsClient struct {
	route53Client route53iface.Route53API
}

// Config contains the AWS settings
type Config struct {
	// AccessKeyID is the AWS access key identifier.
	AccessKeyID string
	// SecretAccessKey is the AWS secret access key.
	SecretAccessKey string
}

func NewClient(credentials Config, region string) (Client, error) {
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
	return &awsClient{
		route53Client: route53.New(sess),
	}, nil
}

func (client *awsClient) ListHostedZonesByNameInput(dnsName string) (*route53.ListHostedZonesByNameOutput, error) {
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

func (client *awsClient) ChangeResourceRecordSets(dnsName string, recordChangeBatch *route53.ChangeBatch) (*route53.ChangeResourceRecordSetsOutput, error) {
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

	// TODO Check status is changed to INSYNC
	recordSetsOutput, err := client.route53Client.ChangeResourceRecordSets(recordChanges)

	if err != nil {
		awsErr := err.(awserr.Error)
		if awsErr.Code() == "InvalidChangeBatch" {
			recordSetNotFound := strings.Contains(awsErr.Message(), "but it was not found")
			if !recordSetNotFound {
				// Kafka cluster failed to create on the cluster, we have an entry in the database.
				recordSetNotFound = strings.Contains(awsErr.Message(), "Domain name is empty")
			}
			if recordSetNotFound {
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
		return errors.BadRequest.UserWrapf(err, msg)
	default:
		return err
	}
}
