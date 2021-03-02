package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/config"
	ocmErrors "github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/services"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/test/mocks"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	testClusterPath = "test_cluster.json"
)

// GetRunningOsdClusterID - is used by tests to get a ClusterID value of an existing OSD cluster.
// If executed against real OCM client, content of /test/integration/test_cluster.json file (if present) is read
// to determine if there is a cluster running and new entry is added to the clusters table.
// If no such file is present, new cluster is created and its clusterID is retrieved.
func GetRunningOsdClusterID(h *test.Helper, t *testing.T) (string, *ocmErrors.ServiceError) {
	return GetOsdClusterID(h, t, true)
}

func GetOsdClusterID(h *test.Helper, t *testing.T, waitForReadiness bool) (string, *ocmErrors.ServiceError) {
	var clusterID string
	if h.Env().Config.OCM.MockMode != config.MockModeEmulateServer && fileExists(testClusterPath, t) {
		clusterID, _ = readClusterDetailsFromFile(h, t)
	}
	if h.Env().Config.OCM.MockMode == config.MockModeEmulateServer {
		_, err := h.Env().Services.Cluster.Create(&api.Cluster{
			CloudProvider: mocks.MockCluster.CloudProvider().ID(),
			Region:        mocks.MockCluster.Region().ID(),
			MultiAZ:       mocks.MockMultiAZ,
		})
		if err != nil {
			return "", err
		}
	}

	if clusterID == "" {
		foundCluster, svcErr := h.Env().Services.Cluster.FindCluster(services.FindClusterCriteria{
			Region:   mocks.MockCluster.Region().ID(),
			Provider: mocks.MockCluster.CloudProvider().ID(),
		})
		if svcErr != nil {
			return "", svcErr
		}
		if foundCluster == nil {
			return "", nil
		} else {
			clusterID = foundCluster.ClusterID
		}
		if err := wait.PollImmediate(10*time.Second, 120*time.Minute, func() (bool, error) {
			foundCluster, err := h.Env().Services.Cluster.FindClusterByID(clusterID)
			if err != nil {
				return true, err
			}
			if foundCluster == nil {
				return false, nil
			}
			return !waitForReadiness || foundCluster.Status.String() == api.ClusterReady.String(), nil
		}); err != nil {
			return "", ocmErrors.GeneralError("Unable to get OSD cluster")
		}
		if h.Env().Config.OCM.MockMode != config.MockModeEmulateServer {
			err := PersistClusterStruct(*foundCluster)
			if err != nil {
				t.Log(fmt.Sprintf("Unable to persist struct for cluster: %s", foundCluster.ID))
			}
		}
	}
	return clusterID, nil
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
		if err := dbConn.Save(cluster).Error; err != nil {
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
