package integration

import (
	"context"
	"fmt"
	"github.com/antihax/optional"
	constants2 "github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/constants"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/dbapi"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/internal/api/public"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/internal/dinosaur/test/mocks/fleetshardsync"
	"github.com/bf2fc6cc711aee1a0c2a/fleet-manager/test/mocks"
	"github.com/bxcodec/faker/v3"
	. "github.com/onsi/gomega"
	"testing"
)

const (
	mockDinosaurName1           = "test-dinosaur1"
	mockDinosaurName2           = "a-dinosaur1"
	mockDinosaurName3           = "z-dinosaur1"
	mockDinosaurName4           = "b-dinosaur1"
	nonExistentDinosaurName     = "non-existentDinosaur"
	nonExistentColumnName    = "non_existentColumn"
	sqlDeleteQuery           = "delete * from clusters;"
	usernameWithSpecialChars = "special+dinosaur@example.com"
	orgId                    = "13640203"
)

type testEnv struct {
	client   *public.APIClient
	ctx      context.Context
	teardown func()
}

func setUp(t *testing.T) *testEnv {
	// create a mock ocm api server, keep all endpoints as defaults
	// see the mocks package for more information on the configurable mock server
	ocmServer := mocks.NewMockConfigurableServerBuilder().Build()

	// setup the test environment, if OCM_ENV=integration then the ocmServer provided will be used instead of actual
	// ocm
	h, client, teardown := test.NewDinosaurHelper(t, ocmServer)

	mockFleetshardSyncBuilder := fleetshardsync.NewMockFleetshardSyncBuilder(h, t)
	mockFleetshardSync := mockFleetshardSyncBuilder.Build()
	mockFleetshardSync.Start()

	// setup pre-requisites to performing requests
	account := h.NewAccount(usernameWithSpecialChars, faker.Name(), faker.Email(), orgId)
	ctx := h.NewAuthenticatedContext(account, nil)

	db := test.TestServices.DBFactory.New()
	dinosaurs := []*dbapi.DinosaurRequest{
		{
			MultiAZ:        false,
			Owner:          usernameWithSpecialChars,
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  mocks.MockCluster.CloudProvider().ID(),
			Name:           mockDinosaurName1,
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusReady.String(),
		},
		{
			MultiAZ:        false,
			Owner:          usernameWithSpecialChars,
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  mocks.MockCluster.CloudProvider().ID(),
			Name:           mockDinosaurName2,
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusReady.String(),
		},
		{
			MultiAZ:        false,
			Owner:          usernameWithSpecialChars,
			Region:         mocks.MockCluster.Region().ID(),
			CloudProvider:  mocks.MockCluster.CloudProvider().ID(),
			Name:           mockDinosaurName3,
			OrganisationId: orgId,
			Status:         constants2.DinosaurRequestStatusReady.String(),
		},
	}

	if err := db.Create(&dinosaurs).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
	}

	if err := db.Find(&dinosaurs).Error; err != nil {
		Expect(err).NotTo(HaveOccurred())
	}

	env := testEnv{
		client: client,
		ctx:    ctx,
		teardown: func() {
			for _, k := range dinosaurs {
				db.Delete(k)
			}
			ocmServer.Close()
			teardown()
			mockFleetshardSync.Stop()
		},
	}
	return &env
}

// Test_DinosaurListSearchAndOrderBy tests getting dinosaur requests list
func Test_DinosaurListSearchAndOrderBy(t *testing.T) {
	env := setUp(t)
	defer env.teardown()

	testCases := []struct {
		name           string
		searchOpts     *public.GetDinosaursOpts
		wantErr        bool
		expectedErr    string
		expectedSize   int32
		expectedTotal  int32
		expectedOrder  []string
		notContains    []string
		validateResult func(list *public.DinosaurRequestList) error
	}{
		{
			name:          "Order By Name Asc",
			searchOpts:    &public.GetDinosaursOpts{OrderBy: optional.NewString("name asc")},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
			expectedOrder: []string{mockDinosaurName2, mockDinosaurName1, mockDinosaurName3},
		},
		{
			name:          "Order By Name Desc",
			searchOpts:    &public.GetDinosaursOpts{OrderBy: optional.NewString("name desc")},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
			expectedOrder: []string{mockDinosaurName3, mockDinosaurName1, mockDinosaurName2},
		},
		{
			name:          "Filter By Name",
			searchOpts:    &public.GetDinosaursOpts{Search: optional.NewString(fmt.Sprintf("name = %s", mockDinosaurName1))},
			wantErr:       false,
			expectedSize:  1,
			expectedTotal: 1,
			expectedOrder: []string{mockDinosaurName1},
		},
		{
			name:          "Filter By Name Not In",
			searchOpts:    &public.GetDinosaursOpts{Search: optional.NewString(fmt.Sprintf("name = %s", nonExistentDinosaurName))},
			wantErr:       false,
			expectedSize:  0,
			expectedTotal: 0,
		},
		{
			name:          "Filter By Name Not Equal",
			searchOpts:    &public.GetDinosaursOpts{Search: optional.NewString(fmt.Sprintf("name <> %s", mockDinosaurName1))},
			wantErr:       false,
			expectedSize:  2,
			expectedTotal: 2,
			notContains:   []string{mockDinosaurName1},
		},
		{
			name:          "Filter By Name Not Equal Non Existent Name",
			searchOpts:    &public.GetDinosaursOpts{Search: optional.NewString(fmt.Sprintf("name <> %s", nonExistentDinosaurName))},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
			notContains:   []string{nonExistentDinosaurName},
		},
		{
			name:          "Filter By Owner With Special Characters",
			searchOpts:    &public.GetDinosaursOpts{Search: optional.NewString(fmt.Sprintf("owner = %s", usernameWithSpecialChars))},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
			notContains:   []string{nonExistentDinosaurName},
			validateResult: func(list *public.DinosaurRequestList) error {
				if list.Items[0].Owner != usernameWithSpecialChars {
					return fmt.Errorf("expecting owner to be '%s' but found '%s'", usernameWithSpecialChars, list.Items[0].Owner)
				}
				return nil
			},
		},
		{
			name:       "Filter By Invalid Column Name",
			searchOpts: &public.GetDinosaursOpts{Search: optional.NewString(fmt.Sprintf("%s <> %s", nonExistentColumnName, mockDinosaurName1))},
			wantErr:    true,
		},
		{
			name:       "Filter By Incomplete Query",
			searchOpts: &public.GetDinosaursOpts{Search: optional.NewString("name <>")},
			wantErr:    true,
		},
		{
			name:       "Filter By Incomplete Composed Query",
			searchOpts: &public.GetDinosaursOpts{Search: optional.NewString(fmt.Sprintf("name <> %s and %s", mockDinosaurName1, sqlDeleteQuery))},
			wantErr:    true,
		},
		{
			name:       "Filter By Invalid Join",
			searchOpts: &public.GetDinosaursOpts{Search: optional.NewString(fmt.Sprintf("name <> %s or_maybe name = %s", nonExistentDinosaurName, mockDinosaurName1))},
			wantErr:    true,
		},
		{
			name:          "Filter By Or Join",
			searchOpts:    &public.GetDinosaursOpts{Search: optional.NewString(fmt.Sprintf("name <> %s or name = %s", nonExistentDinosaurName, mockDinosaurName1))},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
		},
		{
			name:          "Filter By And Join",
			searchOpts:    &public.GetDinosaursOpts{Search: optional.NewString(fmt.Sprintf("name <> %s and cloud_provider = %s", nonExistentDinosaurName, mocks.MockCluster.CloudProvider().ID()))},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
		},
		{
			name:          "Filter By Two Joins",
			searchOpts:    &public.GetDinosaursOpts{Search: optional.NewString(fmt.Sprintf("name <> %s and cloud_provider = %s and region = %s", nonExistentDinosaurName, mocks.MockCluster.CloudProvider().ID(), mocks.MockCluster.Region().ID()))},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
		},
		{
			name: "Filter By Too Many Joins",
			searchOpts: &public.GetDinosaursOpts{Search: optional.NewString(
				fmt.Sprintf(
					"name <> %s and name = %s and name = %s and name = %s and name = %s and name = %s or name <> %s and name = %s and name = %s and name = %s and name = %s and name = %s",
					nonExistentDinosaurName,
					mockDinosaurName1,
					mockDinosaurName1,
					mockDinosaurName1,
					mockDinosaurName1,
					mockDinosaurName1,
					nonExistentDinosaurName,
					mockDinosaurName1,
					mockDinosaurName1,
					mockDinosaurName1,
					mockDinosaurName1,
					mockDinosaurName1))},
			wantErr: true,
		},
		{
			name:          "Filter By Like (%xxx%)",
			searchOpts:    &public.GetDinosaursOpts{Search: optional.NewString("name LIKE %dinosaur1")},
			wantErr:       false,
			expectedSize:  3,
			expectedTotal: 3,
		},
		{
			name:        "MGDSTRM-2956 - Order By Invalid Fields",
			searchOpts:  &public.GetDinosaursOpts{Search: optional.NewString("( SELECT generate_series(1,4);)--")},
			wantErr:     true,
			expectedErr: "400 Bad Request",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RegisterTestingT(t)
			list, _, err := env.client.DefaultApi.GetDinosaurs(env.ctx, tc.searchOpts)
			if tc.wantErr {
				Expect(err).To(HaveOccurred(), "Error wantErr: %v : %v", tc.wantErr, err)

				if tc.expectedErr != "" {
					Expect(err.Error()).To(Equal(tc.expectedErr))
				}
			} else {
				Expect(err).NotTo(HaveOccurred(), "Error wantErr: %v : %v", tc.wantErr, err)
			}

			if err == nil {
				Expect(list.Size).To(Equal(tc.expectedSize))
				Expect(list.Total).To(Equal(tc.expectedTotal))

				if tc.validateResult != nil {
					err := tc.validateResult(&list)
					Expect(err).ToNot(HaveOccurred(), "Returned list didn't pass validation: %v", err)
				}

				for i := 0; i < len(tc.expectedOrder); i++ {
					Expect(list.Items[i].Name).To(Equal(tc.expectedOrder[i]))
				}

				for i := 0; i < len(tc.notContains); i++ {
					for _, item := range list.Items {
						Expect(item.Name).NotTo(Equal(tc.notContains[i]))
					}
				}
			}
		})
	}
}
