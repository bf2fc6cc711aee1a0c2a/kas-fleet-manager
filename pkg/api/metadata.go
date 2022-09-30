/*
Copyright (c) 2018 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// This file contains functions that simplify generation of API metadata.

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/errors"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	"github.com/getsentry/sentry-go"
	"github.com/golang/glog"
)

// getCommitSha() returns the current version (vcs.revision attribute of BuildInfo struct) using a built-in debug package and its ReadBuildInfo() function.
// Upon error, an empty string is returned along with a GeneralError.
// Empty values returned when vsc.revision is empty/unavailable.
func getCommitSha() (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", errors.GeneralError("failed to retrieve commitSha")
	}
	for _, i := range info.Settings {
		if i.Key == "vcs.revision" {
			return i.Value, nil
		}
	}
	return "", nil
}

// ServeHTTP sends API documentation response.
func (m *Metadata) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Set the content type:
	w.Header().Set("Content-Type", "application/json")

	// Prepare the body:
	body := Metadata{
		ID:   m.ID,
		Kind: "API",
		HREF: r.URL.Path,
	}
	for _, v := range m.Versions {
		href := v.HREF
		if href == "" {
			href = v.ID
		}
		body.Versions = append(body.Versions, VersionMetadata{
			ID:   v.ID,
			Kind: "APIVersion",
			HREF: fmt.Sprintf("%s/%s", r.URL.Path, href),
		})
	}

	data, err := json.Marshal(body)
	if err != nil {
		SendPanic(w, r)
		return
	}

	// Send the response:
	_, err = w.Write(data)
	if err != nil {
		err = fmt.Errorf("can't send response body for request '%s'", r.URL.Path)
		glog.Error(err)
		sentry.CaptureException(err)
		return
	}
}

// ServeHTTP sends API version v1 documentation response.
func (v *VersionMetadata) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Get the version:
	version, err := getCommitSha()
	if err != nil {
		shared.HandleError(r, w, err.(*errors.ServiceError))
		return
	}

	// Set the content type:
	w.Header().Set("Content-Type", "application/json")

	// Prepare the body:
	body := VersionMetadata{
		ID:            v.ID,
		Kind:          "APIVersion",
		HREF:          r.URL.Path,
		ServerVersion: version,
	}
	for _, c := range v.Collections {
		href := c.HREF
		if href == "" {
			href = c.ID
		}
		body.Collections = append(body.Collections, CollectionMetadata{
			ID:   c.ID,
			Kind: c.Kind,
			HREF: fmt.Sprintf("%s/%s", r.URL.Path, href),
		})
	}

	data, err := json.Marshal(body)
	if err != nil {
		SendPanic(w, r)
		return
	}

	// Send the response:
	_, err = w.Write(data)
	if err != nil {
		glog.Errorf("can't send response body for request '%s'", r.URL.Path)
		return
	}
}
