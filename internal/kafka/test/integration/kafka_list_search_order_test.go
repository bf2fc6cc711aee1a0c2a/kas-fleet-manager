package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test"
	mockkafka "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kafkas"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/kafka/test/mocks/kasfleetshardsync"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"

	"github.com/antihax/optional"
	"github.com/bxcodec/faker/v3"
	"github.com/onsi/gomega"
)

const (
	mockKafkaName1           = "test-kafka1"
	mockKafkaName2           = "a-kafka1"
	mockKafkaName3           = "z-kafka1"
	nonExistentKafkaName     = "non-existentKafka"
	nonExistentColumnName    = "non_existentColumn"
	sqlDeleteQuery           = "delete * from clusters;"
	usernameWithSpecialChars = "special+kafka@example.com"
	orgId                    = "13640203"
)

type testEnv struct {
	client   *public.APIClient
	ctx      context.Context
	teardown func()
}

func setUp(t *testing.T) *testEnv {
	g := gomega.NewWithT(t)

	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewKafkaHelper(t, ocmServer)

	mockKasFleetshardSyncBuilder := kasfleetshardsync.NewMockKasFleetshardSyncBuilder(h, t)
	mockKasfFleetshardSync := mockKasFleetshardSyncBuilder.Build()
	mockKasfFleetshardSync.Start()

	// setup pre-requisites to performing requests
	account := h.NewAccount(usernameWithSpecialChars, faker.Name(), faker.Email(), orgId)
	ctx := h.NewAuthenticatedContext(account, nil)

	db := test.TestServices.DBFactory.New()
	kafkas := []*dbapi.KafkaRequest{
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.OWNER, usernameWithSpecialChars),
			mockkafka.With(mockkafka.NAME, mockKafkaName1),
		),
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.OWNER, usernameWithSpecialChars),
			mockkafka.With(mockkafka.NAME, mockKafkaName2),
		),
		mockkafka.BuildKafkaRequest(
			mockkafka.WithPredefinedTestValues(),
			mockkafka.With(mockkafka.OWNER, usernameWithSpecialChars),
			mockkafka.With(mockkafka.NAME, mockKafkaName3),
		),
	}

	if err := db.Create(&kafkas).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
	}

	if err := db.Find(&kafkas).Error; err != nil {
		g.Expect(err).NotTo(gomega.HaveOccurred())
	}

	env := testEnv{
		client: client,
		ctx:    ctx,
		teardown: func() {
			for _, k := range kafkas {
				db.Delete(k)
			}
			ocmServer.Close()
			teardown()
			mockKasfFleetshardSync.Stop()
		},
	}
	return &env
}

// Test_KafkaListSearchAndOrderBy tests getting kafka requests list
func Test_KafkaListSearchAndOrderBy(t *testing.T) {
	env := setUp(t)
	defer env.teardown()

	testCases := []struct {
		name           string
		searchOpts     *public.GetKafkasOpts
		wantErr        bool
		expectedErr    string
		expectedSize   int32
		expectedTotal  int32
		expectedOrder  []string
		notContains    []string
		validateResult func(list *public.KafkaRequestList) error
	}{
		{
			name:          "Order By Name Asc",
			searchOpts:    &public.GetKafkasOpts{OrderBy: optional.NewString("name asc")},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
			expectedOrder: []string{mockKafkaName2, mockKafkaName1, mockKafkaName3},
		},
		{
			name:          "Order By Name Desc",
			searchOpts:    &public.GetKafkasOpts{OrderBy: optional.NewString("name desc")},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
			expectedOrder: []string{mockKafkaName3, mockKafkaName1, mockKafkaName2},
		},
		{
			name:          "Filter By Name",
			searchOpts:    &public.GetKafkasOpts{Search: optional.NewString(fmt.Sprintf("name = %s", mockKafkaName1))},
			wantErr:       false,
			expectedSize:  1,
			expectedTotal: 1,
			expectedOrder: []string{mockKafkaName1},
		},
		{
			name:          "Filter By Name Not In",
			searchOpts:    &public.GetKafkasOpts{Search: optional.NewString(fmt.Sprintf("name = %s", nonExistentKafkaName))},
			wantErr:       false,
			expectedSize:  0,
			expectedTotal: 0,
		},
		{
			name:          "Filter By Name Not Equal",
			searchOpts:    &public.GetKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s", mockKafkaName1))},
			wantErr:       false,
			expectedSize:  2,
			expectedTotal: 2,
			notContains:   []string{mockKafkaName1},
		},
		{
			name:          "Filter By Name Not Equal Non Existent Name",
			searchOpts:    &public.GetKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s", nonExistentKafkaName))},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
			notContains:   []string{nonExistentKafkaName},
		},
		{
			name:          "Filter By Owner With Special Characters",
			searchOpts:    &public.GetKafkasOpts{Search: optional.NewString(fmt.Sprintf("owner = %s", usernameWithSpecialChars))},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
			notContains:   []string{nonExistentKafkaName},
			validateResult: func(list *public.KafkaRequestList) error {
				if list.Items[0].Owner != usernameWithSpecialChars {
					return fmt.Errorf("expecting owner to be '%s' but found '%s'", usernameWithSpecialChars, list.Items[0].Owner)
				}
				return nil
			},
		},
		{
			name:       "Filter By Invalid Column Name",
			searchOpts: &public.GetKafkasOpts{Search: optional.NewString(fmt.Sprintf("%s <> %s", nonExistentColumnName, mockKafkaName1))},
			wantErr:    true,
		},
		{
			name:       "Filter By Incomplete Query",
			searchOpts: &public.GetKafkasOpts{Search: optional.NewString("name <>")},
			wantErr:    true,
		},
		{
			name:       "Filter By Incomplete Composed Query",
			searchOpts: &public.GetKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s and %s", mockKafkaName1, sqlDeleteQuery))},
			wantErr:    true,
		},
		{
			name:       "Filter By Invalid Join",
			searchOpts: &public.GetKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s or_maybe name = %s", nonExistentKafkaName, mockKafkaName1))},
			wantErr:    true,
		},
		{
			name:          "Filter By Or Join",
			searchOpts:    &public.GetKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s or name = %s", nonExistentKafkaName, mockKafkaName1))},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
		},
		{
			name:          "Filter By And Join",
			searchOpts:    &public.GetKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s and cloud_provider = %s", nonExistentKafkaName, mocks.MockCluster.CloudProvider().ID()))},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
		},
		{
			name:          "Filter By Two Joins",
			searchOpts:    &public.GetKafkasOpts{Search: optional.NewString(fmt.Sprintf("name <> %s and cloud_provider = %s and region = %s", nonExistentKafkaName, mocks.MockCluster.CloudProvider().ID(), mocks.MockCluster.Region().ID()))},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
		},
		{
			name: "Filter By Too Many Joins",
			searchOpts: &public.GetKafkasOpts{Search: optional.NewString(
				fmt.Sprintf(
					"name <> %s and name = %s and name = %s and name = %s and name = %s and name = %s or name <> %s and name = %s and name = %s and name = %s and name = %s and name = %s",
					nonExistentKafkaName,
					mockKafkaName1,
					mockKafkaName1,
					mockKafkaName1,
					mockKafkaName1,
					mockKafkaName1,
					nonExistentKafkaName,
					mockKafkaName1,
					mockKafkaName1,
					mockKafkaName1,
					mockKafkaName1,
					mockKafkaName1))},
			wantErr: true,
		},
		{
			name:          "Filter By Like (%xxx%)",
			searchOpts:    &public.GetKafkasOpts{Search: optional.NewString("name LIKE %kafka1")},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
		},
		{
			name:        "MGDSTRM-2956 - Order By Invalid Fields",
			searchOpts:  &public.GetKafkasOpts{Search: optional.NewString("( SELECT generate_series(1,4);)--")},
			wantErr:     true,
			expectedErr: "400 Bad Request",
		},
	}

	for _, testcase := range testCases {
		tc := testcase
		t.Run(tc.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			list, resp, err := env.client.DefaultApi.GetKafkas(env.ctx, tc.searchOpts)
			if resp != nil {
				resp.Body.Close()
			}
			if tc.wantErr {
				g.Expect(err).To(gomega.HaveOccurred(), "Error wantErr: %v : %v", tc.wantErr, err)

				if tc.expectedErr != "" {
					g.Expect(err.Error()).To(gomega.Equal(tc.expectedErr))
				}
			} else {
				g.Expect(err).NotTo(gomega.HaveOccurred(), "Error wantErr: %v : %v", tc.wantErr, err)
			}

			if err == nil {
				g.Expect(list.Size).To(gomega.Equal(tc.expectedSize))
				g.Expect(list.Total).To(gomega.Equal(tc.expectedTotal))

				if tc.validateResult != nil {
					err := tc.validateResult(&list)
					g.Expect(err).ToNot(gomega.HaveOccurred(), "Returned list didn't pass validation: %v", err)
				}

				for i := 0; i < len(tc.expectedOrder); i++ {
					g.Expect(list.Items[i].Name).To(gomega.Equal(tc.expectedOrder[i]))
				}

				for i := 0; i < len(tc.notContains); i++ {
					for _, item := range list.Items {
						g.Expect(item.Name).NotTo(gomega.Equal(tc.notContains[i]))
					}
				}
			}
		})
	}
}
