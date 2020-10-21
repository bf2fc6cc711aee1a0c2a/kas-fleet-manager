package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"gitlab.cee.redhat.com/service/managed-services-api/pkg/api"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/config"
	ocmErrors "gitlab.cee.redhat.com/service/managed-services-api/pkg/errors"
	"gitlab.cee.redhat.com/service/managed-services-api/pkg/services"
	"gitlab.cee.redhat.com/service/managed-services-api/test"
	"gitlab.cee.redhat.com/service/managed-services-api/test/mocks"
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
	var clusterID string
	if h.Env().Config.OCM.MockMode != config.MockModeEmulateServer && fileExists(testClusterPath, t) {
		clusterID, _ = readClusterDetailsFromFile(h, t)
	}

	if clusterID == "" {
		foundCluster, svcErr := h.Env().Services.Cluster.FindCluster(services.FindClusterCriteria{
			Region:   mocks.MockCluster.Region().Name(),
			Provider: mocks.MockCluster.CloudProvider().Name(),
		})
		if svcErr != nil {
			return "", svcErr
		}
		if foundCluster == nil {
			foundCluster, svcErr := h.Env().Services.Cluster.Create(&api.Cluster{
				CloudProvider: mocks.MockCloudProvider.ID(),
				ClusterID:     mocks.MockCluster.ID(),
				ExternalID:    mocks.MockCluster.ExternalID(),
				Region:        mocks.MockCluster.Region().ID(),
			})
			if svcErr != nil {
				return "", svcErr
			}
			if foundCluster == nil {
				return "", ocmErrors.GeneralError("Unable to create new OSD cluster")
			}
			clusterID = foundCluster.ID()
		} else {
			clusterID = foundCluster.ClusterID
		}
		if err := wait.PollImmediate(30*time.Second, 120*time.Minute, func() (bool, error) {
			foundCluster, err := h.Env().Services.Cluster.FindClusterByID(clusterID)
			if err != nil {
				return true, err
			}
			return foundCluster.Status.String() == api.ClusterReady.String(), nil
		}); err != nil {
			return "", ocmErrors.GeneralError("Unable to get OSD cluster")
		}
		if h.Env().Config.OCM.MockMode != config.MockModeEmulateServer {
			PersistClusterStruct(*foundCluster)
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
