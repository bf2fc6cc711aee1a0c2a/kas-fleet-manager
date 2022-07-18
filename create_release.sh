#!/bin/bash -ex

git clone -c http.sslVerify=false "https://gitlab-ci-token:$GITLAB_TOKEN@gitlab.cee.redhat.com/mk-ci-cd/managed-kafka-versions.git" managed-kafka-versions
cd managed-kafka-versions
BRANCH_NAME="kas-fleet-manager-${VERSION}"
# only update the config, if different
CURRENT_COMMIT_SHA=$(yq '.service.scm.commitSha' services/kas-fleet-manager.yaml)
echo "Checking if the latest commit sha: $LATEST_COMMIT is different than current config commit sha: $CURRENT_COMMIT_SHA"
if [[ "${CURRENT_COMMIT_SHA}" != "${LATEST_COMMIT}" ]]; then
  git checkout -b "$BRANCH_NAME"
  echo "Updating commit sha and image tag for kas-fleet-manager configuration"
  # update commitSha
  yq -i ".service.scm.commitSha = \"$LATEST_COMMIT\"" services/kas-fleet-manager.yaml
  # update image tag
  yq -i ".service.image.tag = \"$VERSION\"" services/kas-fleet-manager.yaml
  git config user.name "${AUTHOR_NAME}"
  git config user.email "${AUTHOR_EMAIL}"
  git commit -a -m "kas-fleet-manager stage release $VERSION"
  # create an MR in managed-kafka-versions repo
  echo "Creating MR with new config in managed-kafka-versions repository"
  git push --force -o merge_request.create="true" -o merge_request.title="kas-fleet-manager stage release $VERSION" -o merge_request.description="https://gitlab.cee.redhat.com/service/kas-fleet-manager/-/compare/$CURRENT_COMMIT_SHA...$LATEST_COMMIT" -o merge_request.merge_when_pipeline_succeeds="false" -u origin "$BRANCH_NAME"
else
  echo "No new version detected for kas-fleet-manager"
fi
