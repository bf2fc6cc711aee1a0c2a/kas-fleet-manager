package aws

import (
	"github.com/aws/aws-sdk-go/service/route53/route53iface"
)

// We create a type alias to route53iface.Route53API to be able to autogenerate
// mocks for it
//
//go:generate moq -out route53_client_moq.go . route53Client
type route53Client = route53iface.Route53API
