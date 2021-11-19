package api

import (
	"strings"

	semver "github.com/blang/semver/v4"
)

// buildAwareSemanticVersioningCompare compares v1 and v2 strings as
// semantic versions (https://semver.org/) with the added behavior of
// comparing the build metadata information in the version too.
// If v1 is smaller than v2 a -1 is returned
// If v1 is greater than v2 1 is returned
// If v1 x.y.z and pre-release elements are equal to v2 x.y.z and pre-release
// elements then a lexicographical comparison between the build metadata
// elements is performed:
//  - If v1's build metadata is smaller than v2's build metadata -1 is returned
//  - If v1's build metadata is greater than v2's build metadata 1 is returned
//  - If v2 and v2's metadata are equal then 0 is returned
// An error is returned if the provided version strings cannot be interpreted
// as semantic versioning strings
func buildAwareSemanticVersioningCompare(v1, v2 string) (int, error) {
	v1Semver, err := semver.ParseTolerant(v1)
	if err != nil {
		return 0, err
	}

	v2Semver, err := semver.ParseTolerant(v2)
	if err != nil {
		return 0, err
	}

	res := v1Semver.Compare(v2Semver)
	if res == 0 {
		v1BuildVersion := strings.Join(v1Semver.Build[:], ".")
		v2BuildVersion := strings.Join(v2Semver.Build[:], ".")
		if v1BuildVersion == v2BuildVersion {
			res = 0
		} else if v1BuildVersion > v2BuildVersion {
			res = 1
		} else {
			res = -1
		}
	}

	return res, nil
}
