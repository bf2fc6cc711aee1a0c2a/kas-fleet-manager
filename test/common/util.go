package common

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	. "github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api/private/openapi"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	ocmErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/ocm"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	clustersmgmtv1 "github.com/openshift-online/ocm-sdk-go/clustersmgmt/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	testClusterPath = "test_cluster.json"
)

func waitForClusterStatus(h *test.Helper, t *testing.T, clusterID string, expectedStatus api.ClusterStatus) error {
	if expectedStatus == api.ClusterReady {
		clusterReadyInterval := 2 * time.Minute
		clusterReadyTimeout := 3 * time.Hour
		if h.Env().Config.OCM.EnableMock {
			clusterReadyInterval = 1 * time.Second
			clusterReadyTimeout = 5 * time.Minute
		}
		if err := WaitForAndUpdateOSDClusterStatusToReady(h, t, clusterID, clusterReadyInterval, clusterReadyTimeout); err != nil {
			return err
		}
	}
	return wait.PollImmediate(1*time.Second, 120*time.Minute, func() (bool, error) {
		foundCluster, err := h.Env().Services.Cluster.FindClusterByID(clusterID)
		if err != nil {
			return true, err
		}
		if foundCluster == nil {
			return false, nil
		}
		return foundCluster.Status.String() == expectedStatus.String(), nil
	})
}

func Poll(interval time.Duration, timeout time.Duration,
	onRetry func(attempt int, maxRetries int) (done bool, err error),
	onStart func(maxRetries int) error,
	onFinish func(attempt int, maxRetries int, err error) error) error {
	if onRetry == nil {
		return fmt.Errorf("no retry handler has been specified")
	}

	maxAttempts := int(timeout / interval)

	if onStart != nil {
		if err := onStart(maxAttempts); err != nil {
			return err
		}
	}

	attempt := 0
	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		attempt++
		return onRetry(attempt, maxAttempts)
	})

	if onFinish != nil {
		return onFinish(attempt, maxAttempts, err)
	}

	return err
}

// GetRunningOsdClusterID - is used by tests to get a ClusterID value of an existing OSD cluster in a 'ready' state.
// If executed against real OCM client, content of /test/integration/test_cluster.json file (if present) is read
// to determine if there is a cluster running and new entry is added to the clusters table.
// If no such file is present, new cluster is created and its clusterID is retrieved.
func GetRunningOsdClusterID(h *test.Helper, t *testing.T) (string, *ocmErrors.ServiceError) {
	return GetOSDClusterIDAndWaitForStatus(h, t, api.ClusterReady)
}

func GetOSDClusterIDAndWaitForStatus(h *test.Helper, t *testing.T, expectedStatus api.ClusterStatus) (string, *ocmErrors.ServiceError) {
	return GetOSDClusterID(h, t, &expectedStatus)
}

// GetOSDClusterID - is used by tests to get a ClusterID value of an existing OSD cluster.
// If executed against real OCM client, content of /test/integration/test_cluster.json file (if present) is read
// to determine if there is a cluster running and new entry is added to the clusters table.
// If no such file is present, new cluster is created and its clusterID is retrieved.
// If expectedStatus is not nil then it will wait until expectedStatus is reached.
func GetOSDClusterID(h *test.Helper, t *testing.T, expectedStatus *api.ClusterStatus) (string, *ocmErrors.ServiceError) {
	var foundCluster *api.Cluster
	var err *ocmErrors.ServiceError

	if h.Env().Config.OCM.MockMode != config.MockModeEmulateServer && fileExists(testClusterPath, t) {
		clusterID, _ := readClusterDetailsFromFile(h, t)
		foundCluster, err = h.Env().Services.Cluster.FindClusterByID(clusterID)
		if err != nil {
			return "", err
		}
	}

	if h.Env().Config.OCM.MockMode == config.MockModeEmulateServer {
		// first check if there is an existing valid cluster that we can use
		foundCluster, err = findFirstValidCluster(h)
		if err != nil {
			return "", err
		}
	}

	// No existing cluster found
	if foundCluster == nil {
		// create a new cluster
		_, err := h.Env().Services.Cluster.Create(&api.Cluster{
			CloudProvider: mocks.MockCluster.CloudProvider().ID(),
			Region:        mocks.MockCluster.Region().ID(),
			MultiAZ:       mocks.MockMultiAZ,
		})
		if err != nil {
			return "", err
		}
		// need to wait for it to be assigned
		waitErr := wait.PollImmediate(1*time.Second, 5*time.Minute, func() (done bool, err error) {
			c, findErr := findFirstValidCluster(h)
			if findErr != nil {
				return false, findErr
			}
			foundCluster = c
			return foundCluster != nil && foundCluster.ClusterID != "", nil
		})
		if waitErr != nil {
			return "", ocmErrors.GeneralError("error to find a valid cluster: %v", waitErr)
		}

		// Only persist new cluster if executing against real ocm
		if h.Env().Config.OCM.MockMode != config.MockModeEmulateServer {
			pErr := PersistClusterStruct(*foundCluster)
			if pErr != nil {
				t.Log(fmt.Sprintf("Unable to persist struct for cluster: %s", foundCluster.ID))
			}
		}
	}

	if expectedStatus != nil {
		err := waitForClusterStatus(h, t, foundCluster.ClusterID, *expectedStatus)
		if err != nil {
			return "", ocmErrors.GeneralError("error waiting for cluster '%s' to reach '%s': %v", foundCluster.ClusterID, *expectedStatus, err)
		}
	}

	return foundCluster.ClusterID, nil
}

// Waits for the terraforming to be complete and updates the status of the OSD cluster once it's deemed to be 'ready'.
//
// An OSD cluster is deemed 'ready' when the managed-kafka and kas-fleetshard-operator addon have been installed.
// Since the cluster agent cannot talk to the integration tests directly, this sends a request to the /agent-clusters/{id}/status endpoint to update
// the status of the OSD cluster in the database.
func WaitForAndUpdateOSDClusterStatusToReady(h *test.Helper, t *testing.T, clusterID string, interval time.Duration, timeout time.Duration) error {
	ocmClient := ocm.NewClient(h.Env().Clients.OCM.Connection)
	return Poll(interval, timeout, func(attempt int, maxAttempts int) (bool, error) {
		cluster, svcErr := h.Env().Services.Cluster.FindClusterByID(clusterID)
		if svcErr != nil {
			return true, fmt.Errorf("failed to get cluster details (id: %s) from the database: %v", clusterID, svcErr)
		}

		if osdClusterCanProcessStatusReports(cluster.Status) {
			managedKafkaAddon, err := ocmClient.GetAddon(clusterID, api.ManagedKafkaAddonID)
			if err != nil {
				return true, fmt.Errorf("failed to get '%s' addon: %v", api.ManagedKafkaAddonID, err)
			}

			kasFleetShardOperatorAddon, err := ocmClient.GetAddon(clusterID, api.KasFleetshardOperatorAddonId)
			if err != nil {
				return true, fmt.Errorf("failed to get '%s' addon: %v", api.KasFleetshardOperatorAddonId, err)
			}

			if managedKafkaAddon.State() == clustersmgmtv1.AddOnInstallationStateReady &&
				kasFleetShardOperatorAddon.State() == clustersmgmtv1.AddOnInstallationStateReady {
				return true, nil
			}
		}

		t.Logf("%d/%d - Waiting for terraforming to complete for cluster '%s'", attempt, maxAttempts, clusterID)
		return false, nil
	}, func(maxAttempts int) error {
		t.Logf("Waiting for cluster '%s' to be ready. Max Attempts: %d, Interval Duration: %d", clusterID, maxAttempts, interval)
		return nil
	}, func(attempt, maxAttempts int, err error) error {
		if err != nil {
			return err
		}

		t.Logf("%d/%d - Finished waiting for cluster '%s' to be ready", attempt, maxAttempts, clusterID)

		// Call endpoint to update cluster status to 'ready'
		ctx := NewAuthenticatedContextForDataPlaneCluster(h, clusterID)
		privateAPIClient := h.NewPrivateAPIClient()
		clusterStatusUpdateRequest := SampleDataPlaneclusterStatusRequestWithAvailableCapacity()
		if _, err = privateAPIClient.AgentClustersApi.UpdateAgentClusterStatus(ctx, clusterID, *clusterStatusUpdateRequest); err != nil {
			return fmt.Errorf("failed to update cluster status via agent endpoint: %v", err)
		}
		return nil
	})
}

// Returns true/false whether an OSD cluster can process status reports. If false, status updates to the agent cluster endpoint will not be applied.
func osdClusterCanProcessStatusReports(clusterStatus api.ClusterStatus) bool {
	if clusterStatus == api.ClusterReady ||
		clusterStatus == api.ClusterComputeNodeScalingUp ||
		clusterStatus == api.ClusterFull ||
		clusterStatus == api.ClusterWaitingForKasFleetShardOperator {
		return true
	}
	return false
}

func findFirstValidCluster(h *test.Helper) (*api.Cluster, *ocmErrors.ServiceError) {
	foundClusters, svcErr := h.Env().Services.Cluster.FindAllClusters(services.FindClusterCriteria{
		Region:   mocks.MockCluster.Region().ID(),
		Provider: mocks.MockCluster.CloudProvider().ID(),
		MultiAZ:  mocks.MockMultiAZ,
	})
	if svcErr != nil {
		return nil, svcErr
	}
	var foundCluster *api.Cluster
	for _, c := range foundClusters {
		if c.Status.String() != api.ClusterDeprovisioning.String() {
			foundCluster = c
			return foundCluster, nil
		}
	}
	return foundCluster, nil
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string, t *testing.T) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		t.Log(fmt.Sprintf("%s not found", filename))
		return false
	}
	return !info.IsDir()
}

// PersistClusterStruct to json file to be reused by tests that require a running cluster
func PersistClusterStruct(cluster api.Cluster) error {
	// We intentionally always persist the cluster status as ClusterProvisioned
	// with the aim of letting cluster reconciler in other tests reach
	// the real status depending on the configured environment settings
	cluster.Status = api.ClusterProvisioned
	file, err := json.MarshalIndent(cluster, "", " ")
	if err != nil {
		return ocmErrors.GeneralError(fmt.Sprintf("Failed to marshal cluster struct details to a file: %v", err))
	}
	err = ioutil.WriteFile(testClusterPath, file, 0644)
	if err != nil {
		return ocmErrors.GeneralError(fmt.Sprintf("Failed to persist cluster struct details to a file: %v", err))
	}
	return nil
}

// get clusterID from file if exists. Any issues with reading content of the file results in returning
// an empty string with an error. If no file found - return empty string and nil as an error
func readClusterDetailsFromFile(h *test.Helper, t *testing.T) (string, error) {
	if fileExists(testClusterPath, t) {
		file, err := ioutil.ReadFile(testClusterPath)
		if err != nil {
			return "", ocmErrors.GeneralError(fmt.Sprintf("Failed to ReadFile with cluster details: %v", err))
		}

		cluster := &api.Cluster{}

		marshalErr := json.Unmarshal([]byte(file), cluster)
		if marshalErr != nil {
			return "", ocmErrors.GeneralError(fmt.Sprintf("Failed to Unmarshal cluster details from file: %v", marshalErr))
		}

		dbConn := h.Env().DBFactory.New()
		if err := dbConn.FirstOrCreate(cluster, &api.Cluster{ClusterID: cluster.ClusterID}).Error; err != nil {
			return "", ocmErrors.GeneralError(fmt.Sprintf("Failed to save cluster details to database: %v", err))
		}
		return cluster.ClusterID, nil
	}
	return "", nil
}

func getMetrics(t *testing.T) string {
	metricsConfig := config.NewMetricsConfig()
	metricsAddress := metricsConfig.BindAddress
	var metricsURL string
	if metricsConfig.EnableHTTPS {
		metricsURL = fmt.Sprintf("https://%s/metrics", metricsAddress)
	} else {
		metricsURL = fmt.Sprintf("http://%s/metrics", metricsAddress)
	}
	response, err := http.Get(metricsURL)
	if err != nil {
		t.Fatalf("failed to get response from %s: %s", metricsURL, err.Error())
	}
	defer response.Body.Close()

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("failed to read response data: %s", err.Error())
	}

	responseString := string(responseData)
	return responseString
}

// CheckMetricExposed - checks whether metric is exposed in the metrics URL
func CheckMetricExposed(h *test.Helper, t *testing.T, metric string) {
	resp := getMetrics(t)
	Expect(strings.Contains(resp, metric)).To(Equal(true))
}

// CheckMetric - check if the given metric exists in the metrics data
func CheckMetric(h *test.Helper, t *testing.T, metric string, exist bool) {
	resp := getMetrics(t)
	Expect(strings.Contains(resp, metric)).To(Equal(exist))
}

// Returns an authenticated context to be used for calling the data plane endpoints
func NewAuthenticatedContextForDataPlaneCluster(h *test.Helper, clusterID string) context.Context {
	account := h.NewAllowedServiceAccount()
	claims := jwt.MapClaims{
		"iss": h.AppConfig.Keycloak.KafkaRealm.ValidIssuerURI,
		"realm_access": map[string][]string{
			"roles": {"kas_fleetshard_operator"},
		},
		"kas-fleetshard-operator-cluster-id": clusterID,
	}
	token := h.CreateJWTStringWithClaim(account, claims)
	ctx := context.WithValue(context.Background(), openapi.ContextAccessToken, token)

	return ctx
}

// Returns a sample data plane cluster status request with available capacity
func SampleDataPlaneclusterStatusRequestWithAvailableCapacity() *openapi.DataPlaneClusterUpdateStatusRequest {
	return &openapi.DataPlaneClusterUpdateStatusRequest{
		Conditions: []openapi.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
		Total: openapi.DataPlaneClusterUpdateStatusRequestTotal{
			IngressEgressThroughputPerSec: &[]string{"test"}[0],
			Connections:                   &[]int32{1000000}[0],
			DataRetentionSize:             &[]string{"test"}[0],
			Partitions:                    &[]int32{1000000}[0],
		},
		NodeInfo: openapi.DataPlaneClusterUpdateStatusRequestNodeInfo{
			Ceiling:                &[]int32{20}[0],
			Floor:                  &[]int32{3}[0],
			Current:                &[]int32{5}[0],
			CurrentWorkLoadMinimum: &[]int32{3}[0],
		},
		Remaining: openapi.DataPlaneClusterUpdateStatusRequestTotal{
			Connections:                   &[]int32{1000000}[0], // TODO set the values taking the scale-up value if possible or a deterministic way to know we'll pass it
			Partitions:                    &[]int32{1000000}[0],
			IngressEgressThroughputPerSec: &[]string{"test"}[0],
			DataRetentionSize:             &[]string{"test"}[0],
		},
		ResizeInfo: openapi.DataPlaneClusterUpdateStatusRequestResizeInfo{
			NodeDelta: &[]int32{3}[0],
			Delta: &openapi.DataPlaneClusterUpdateStatusRequestResizeInfoDelta{
				Connections:                   &[]int32{10000}[0],
				Partitions:                    &[]int32{10000}[0],
				IngressEgressThroughputPerSec: &[]string{"test"}[0],
				DataRetentionSize:             &[]string{"test"}[0],
			},
		},
	}
}
