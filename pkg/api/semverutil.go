package api

import semver "github.com/blang/semver/v4"

// semanticVersioningCompare compares v1 and v2 strings as
// semantic versions (https://semver.org/). If v1 is smaller than v2 a -1 is
// returned. If v1 is equal to v2 0 is returned. If v1 is greater than v2 1 is
// returned.
// An error is returned if the provided version strings cannot be interpreted
// as semantic versioning strings
func semanticVersioningCompare(v1, v2 string) (int, error) {
	v1Semver, err := semver.ParseTolerant(v1)
	if err != nil {
		return 0, err
	}

	v2Semver, err := semver.ParseTolerant(v2)
	if err != nil {
		return 0, err
	}

	return v1Semver.Compare(v2Semver), nil
}
