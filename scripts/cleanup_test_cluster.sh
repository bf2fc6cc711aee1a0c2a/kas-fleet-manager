#!/bin/bash

# This bash script is used to clean up an OSD cluster created and used by integration tests
# against real OCM environment. It uses a /test/integration/test_cluster.json file with cluster details created by the tests.

FILE=test/integration/test_cluster.json
INTEGRATION_ENV="integration"
ENV=$(echo "$OCM_ENV")
if test -f "$FILE"; then
  if [[ $ENV != "$INTEGRATION_ENV" ]] ; then
    clusterID=$(jq -r .cluster_id ${FILE})
    if [[ $clusterID != "null" ]] ; then
      ocm login --url=https://api.stage.openshift.com/ --token="$OCM_OFFLINE_TOKEN"
      ocm delete /api/clusters_mgmt/v1/clusters/"$clusterID"
      rm $FILE
    fi
  fi
fi
