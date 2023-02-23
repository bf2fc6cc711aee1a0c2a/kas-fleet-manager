package testutils

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/constants"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/kafkas/types"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/utils"
	"github.com/google/uuid"
)

var makeFunc = utils.MakeFunctionalParameter[string, dbapi.KafkaRequest]

var WithID = makeFunc("ID")
var WithStatus = makeFunc("Status")
var WithInstanceType = makeFunc("InstanceType")
var WithSizeID = makeFunc("SizeId")
var WithActualBillingModel = makeFunc("ActualKafkaBillingModel")
var WithDesiredBillingModel = makeFunc("DesiredKafkaBillingModel")
var WithName = makeFunc("Name")

func WithDefaultTestValues() utils.FunctionalParameter[dbapi.KafkaRequest] {
	return func(request *dbapi.KafkaRequest) {
		request.ID = uuid.New().String()
		request.Name = uuid.New().String()
		request.Status = constants.KafkaRequestStatusAccepted.String()
		request.InstanceType = types.STANDARD.String()
		request.SizeId = "x1"
		request.DesiredKafkaBillingModel = "standard"
		request.ActualKafkaBillingModel = "standard"
	}
}

func NewKafkaRequest(params ...utils.FunctionalParameter[dbapi.KafkaRequest]) *dbapi.KafkaRequest {
	k := &dbapi.KafkaRequest{}
	for _, functionalParam := range params {
		functionalParam(k)
	}
	return k
}
