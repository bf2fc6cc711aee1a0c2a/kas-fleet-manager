#!/bin/bash -ex
#
# Copyright (c) 2019 Red Hat, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# This script is executed by a Jenkins job for each change request. If it
# doesn't succeed the change won't be merged.

# The version of `podman` available in the Jenkins environment doesn't work
# well when multiple sessions are created. The pause process that it creates is
# killed when the session finishes, but the file containing its pid is left
# around. That makes the next execution of `podman` fail because it can't join
# that pause process. To avoid that we need to explicitly tell `podman` and
# related tools to use a different directory to store state, and we also need
# to clean it before starting the build.
export XDG_RUNTIME_DIR="${PWD}/.xdg"
rm -rf "${XDG_RUNTIME_DIR}"
mkdir -p "${XDG_RUNTIME_DIR}"

# Set the `GOBIN` environment variable so that dependencies will be installed
# always in the same place, regardless of the value of `GOPATH`:
export GOBIN="${PWD}/.gobin"
export PATH="${GOBIN}:${PATH}"

test -f go1.13.6.linux-amd64.tar.gz || curl -O -J https://dl.google.com/go/go1.13.6.linux-amd64.tar.gz

export IMAGE_NAME="test/managed-services-api"

docker build -t "$IMAGE_NAME" -f Dockerfile.integration.test .
docker run -i "$IMAGE_NAME"