package integration

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/antihax/optional"
	. "github.com/onsi/gomega"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api/openapi"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	utils "gitlab.cee.redhat.com/service/managed-services-api/test/common"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
)

const (
	mockKafkaName1        = "test-kafka1"
	mockKafkaName2        = "a-kafka1"
	mockKafkaName3        = "z-kafka1"
	nonExistentKafkaName  = "non-existentKafka"
	nonExistentColumnName = "non_existentColumn"
	invalidSearchValue    = "&123abc"
	sqlDeleteQuery        = "delete * from clusters;"
)

// Test_KafkaListSearchAndOrderBy tests getting kafka requests list
func Test_KafkaListSearchAndOrderBy(t *testing.T) {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()
	defer ocmServer.Close()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.RegisterIntegration(t, ocmServer)
	defer teardown()

	// setup pre-requisites to performing requests
	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	// get initial list (should be empty)
	initList, resp, err := client.DefaultApi.ListKafkas(ctx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(initList.Items).To(BeEmpty(), "Expected empty kafka requests list")
	Expect(initList.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(initList.Total).To(Equal(int32(0)), "Expected Total == 0")

	clusterID, getClusterErr := utils.GetRunningOsdClusterID(h, t)
	if getClusterErr != nil {
		t.Fatalf("Failed to retrieve cluster details from persisted .json file: %v", getClusterErr)
	}
	if clusterID == "" {
		panic("No cluster found")
	}

	k := openapi.KafkaRequestPayload{
		Region:        mocks.MockCluster.Region().ID(),
		CloudProvider: mocks.MockCluster.CloudProvider().ID(),
		Name:          mockKafkaName1,
		MultiAz:       true,
	}

	// create three kafka_requests to test search and orderBy
	_, _, err = client.DefaultApi.CreateKafka(ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded KafkaRequest: %s", err.Error())
	}

	k.Name = mockKafkaName2
	_, _, err = client.DefaultApi.CreateKafka(ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded KafkaRequest: %s", err.Error())
	}
	k.Name = mockKafkaName3
	_, _, err = client.DefaultApi.CreateKafka(ctx, true, k)
	if err != nil {
		t.Fatalf("failed to create seeded KafkaRequest: %s", err.Error())
	}

	// get populated list of kafka requests
	populatedKafkaList, _, err := client.DefaultApi.ListKafkas(ctx, nil)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(len(populatedKafkaList.Items)).To(Equal(3), "Expected kafka requests list length to be 1")
	Expect(populatedKafkaList.Size).To(Equal(int32(3)), "Expected Size == 3")
	Expect(populatedKafkaList.Total).To(Equal(int32(3)), "Expected Total == 3")

	// TEST orderBy
	orderBy := &openapi.ListKafkasOpts{OrderBy: optional.NewString("name asc")}

	// test orderBy (by name in ascending order)
	listByNameAsc, _, err := client.DefaultApi.ListKafkas(ctx, orderBy)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(listByNameAsc.Size).To(Equal(int32(3)), "Expected Size == 3")
	Expect(listByNameAsc.Total).To(Equal(int32(3)), "Expected Total == 3")
	Expect(listByNameAsc.Items[0].Name).To(Equal(mockKafkaName2))
	Expect(listByNameAsc.Items[2].Name).To(Equal(mockKafkaName3))

	orderBy = &openapi.ListKafkasOpts{OrderBy: optional.NewString("name desc")}

	// test orderBy (by name in descending order)
	listByNameDesc, _, err := client.DefaultApi.ListKafkas(ctx, orderBy)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(listByNameDesc.Size).To(Equal(int32(3)), "Expected Size == 3")
	Expect(listByNameDesc.Total).To(Equal(int32(3)), "Expected Total == 3")
	Expect(listByNameDesc.Items[2].Name).To(Equal(mockKafkaName2))
	Expect(listByNameDesc.Items[0].Name).To(Equal(mockKafkaName3))

	// TEST search
	// search by name = ?
	search := &openapi.ListKafkasOpts{Search: optional.NewString(fmt.Sprintf("name = %s", mockKafkaName1))}

	searchName, _, err := client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(searchName.Size).To(Equal(int32(1)), "Expected Size == 1")
	Expect(searchName.Total).To(Equal(int32(1)), "Expected Total == 1")
	Expect(searchName.Items[0].Name).To(Equal(mockKafkaName1))

	// search by name = ? with name not in the list
	search = &openapi.ListKafkasOpts{Search: optional.NewString(fmt.Sprintf("name = %s", nonExistentKafkaName))}

	searchName, _, err = client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(searchName.Size).To(Equal(int32(0)), "Expected Size == 0")
	Expect(searchName.Total).To(Equal(int32(0)), "Expected Total == 0")

	// search by name <> ?
	search = &openapi.ListKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s", mockKafkaName1))}

	searchNameInv, _, err := client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(searchNameInv.Size).To(Equal(int32(2)), "Expected Size == 2")
	Expect(searchNameInv.Total).To(Equal(int32(2)), "Expected Total == 2")
	Expect(searchNameInv.Items[0].Name).NotTo(Equal(mockKafkaName1))
	Expect(searchNameInv.Items[1].Name).NotTo(Equal(mockKafkaName1))

	// search by name <> ? with a name that doesn't exist in the kafka list
	search = &openapi.ListKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s", nonExistentKafkaName))}

	searchNameInv, _, err = client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(searchNameInv.Size).To(Equal(int32(3)), "Expected Size == 3")
	Expect(searchNameInv.Total).To(Equal(int32(3)), "Expected Total == 3")
	Expect(searchNameInv.Items[0].Name).NotTo(Equal(nonExistentKafkaName))
	Expect(searchNameInv.Items[1].Name).NotTo(Equal(nonExistentKafkaName))
	Expect(searchNameInv.Items[2].Name).NotTo(Equal(nonExistentKafkaName))

	// search with invalid column name
	search = &openapi.ListKafkasOpts{Search: optional.NewString(fmt.Sprintf("%s <> %s", nonExistentColumnName, mockKafkaName1))}
	_, _, err = client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).To(HaveOccurred()) // expecting an error here

	// search with invalid search value
	search = &openapi.ListKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s", invalidSearchValue))}
	_, _, err = client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).To(HaveOccurred()) // expecting an error here

	// search with incomplete query
	search = &openapi.ListKafkasOpts{Search: optional.NewString("name <>")}
	_, _, err = client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).To(HaveOccurred()) // expecting an error here

	// search with dangerous SQL query (which is also an incomplete query)
	search = &openapi.ListKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s and %s", mockKafkaName1, sqlDeleteQuery))}
	_, _, err = client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).To(HaveOccurred()) // expecting an error here

	// search with invalid join
	search = &openapi.ListKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s or_maybe name = %s", nonExistentKafkaName, mockKafkaName1))}
	_, _, err = client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).To(HaveOccurred()) // expecting an error here

	// search with `or` join
	search = &openapi.ListKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s or name = %s", nonExistentKafkaName, mockKafkaName1))}
	searchJoined, _, err := client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(searchJoined.Size).To(Equal(int32(3)), "Expected Size == 3")
	Expect(searchJoined.Total).To(Equal(int32(3)), "Expected Total == 3")

	// search with 'and' join
	search = &openapi.ListKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s and cloud_provider = %s", nonExistentKafkaName, mocks.MockCluster.CloudProvider().ID()))}
	searchJoined, _, err = client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(searchJoined.Size).To(Equal(int32(3)), "Expected Size == 3")
	Expect(searchJoined.Total).To(Equal(int32(3)), "Expected Total == 3")

	// search with valid query with two joins
	search = &openapi.ListKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s and cloud_provider = %s and region = %s", nonExistentKafkaName, mocks.MockCluster.CloudProvider().ID(), mocks.MockCluster.Region().ID()))}
	searchJoined, _, err = client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(searchJoined.Size).To(Equal(int32(3)), "Expected Size == 3")
	Expect(searchJoined.Total).To(Equal(int32(3)), "Expected Total == 3")

	// search with too many joins
	search = &openapi.ListKafkasOpts{Search: optional.NewString(
		fmt.Sprintf("name <> %s and name = %s and name = %s and name = %s and name = %s and name = %s",
			nonExistentKafkaName,
			mockKafkaName1,
			mockKafkaName1,
			mockKafkaName1,
			mockKafkaName1,
			mockKafkaName1))}
	_, _, err = client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).To(HaveOccurred()) // expecting an error here

	// checking that like works correctly
	search = &openapi.ListKafkasOpts{Search: optional.NewString("name LIKE %kafka1")}
	searchLike, _, err := client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(searchLike.Size).To(Equal(int32(3)), "Expected Size == 3")
	Expect(searchLike.Total).To(Equal(int32(3)), "Expected Total == 3")

	search = &openapi.ListKafkasOpts{Search: optional.NewString("name LIKE test%")}
	searchLike, _, err = client.DefaultApi.ListKafkas(ctx, search)
	Expect(err).NotTo(HaveOccurred(), "Error occurred when attempting to list kafka requests: %v", err)
	Expect(searchLike.Size).To(Equal(int32(1)), "Expected Size == 1")
	Expect(searchLike.Total).To(Equal(int32(1)), "Expected Total == 1")
	Expect(searchNameInv.Items[0].Name).NotTo(Equal(mockKafkaName1))
}
