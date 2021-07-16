package api

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

func TestGetAvailableStrimziVersions(t *testing.T) {
	tests := []struct {
		name    string
		cluster func() *Cluster
		want    []StrimziVersion
		wantErr bool
	}{
		{
			name: "When cluster has a non empty list of available strimzi versions those are returned",
			cluster: func() *Cluster {
				inputStrimziVersions := []StrimziVersion{
					StrimziVersion{Version: "v3", Ready: true},
					StrimziVersion{Version: "v6", Ready: false},
					StrimziVersion{Version: "v7", Ready: true},
				}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want: []StrimziVersion{
				StrimziVersion{Version: "v3", Ready: true},
				StrimziVersion{Version: "v6", Ready: false},
				StrimziVersion{Version: "v7", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When cluster has a non empty list of available strimzi versions in legacy list of string format those are returned",
			cluster: func() *Cluster {
				inputStrimziVersionsInLegacyFormat := []string{"v3", "v6", "v7"}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersionsInLegacyFormat)
				if err != nil {
					panic(err)
				}
				fmt.Println(inputStrimziVersionsInLegacyFormat)
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want: []StrimziVersion{
				StrimziVersion{Version: "v3", Ready: true},
				StrimziVersion{Version: "v6", Ready: true},
				StrimziVersion{Version: "v7", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When cluster has an empty list of available strimzi the empty list is returned",
			cluster: func() *Cluster {
				inputStrimziVersions := []StrimziVersion{}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want:    []StrimziVersion{},
			wantErr: false,
		},
		{
			name: "When cluster has a nil list of available strimzi the empty list is returned",
			cluster: func() *Cluster {
				res := Cluster{AvailableStrimziVersions: nil}
				return &res
			},
			want:    []StrimziVersion{},
			wantErr: false,
		},
		{
			name: "When cluster has an invalid JSON an error is returned",
			cluster: func() *Cluster {
				res := Cluster{AvailableStrimziVersions: []byte(`"keyone": valueone`)}
				return &res
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.cluster().GetAvailableStrimziVersions()
			gotErr := err != nil
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("wantErr: %v got: %v", tt.wantErr, err)
			}
			if !reflect.DeepEqual(res, tt.want) {
				t.Errorf("want: %v got: %v", tt.want, res)
			}
		})
	}
}

func TestGetAvailableAndReadyStrimziVersions(t *testing.T) {
	tests := []struct {
		name    string
		cluster func() *Cluster
		want    []StrimziVersion
		wantErr bool
	}{
		{
			name: "When cluster has a non empty list of available strimzi versions those ready returned",
			cluster: func() *Cluster {
				inputStrimziVersions := []StrimziVersion{
					StrimziVersion{Version: "v3", Ready: true},
					StrimziVersion{Version: "v6", Ready: false},
					StrimziVersion{Version: "v7", Ready: true},
				}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want: []StrimziVersion{
				StrimziVersion{Version: "v3", Ready: true},
				StrimziVersion{Version: "v7", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When cluster has a non empty list of available strimzi versions in legacy list of string format those are returned",
			cluster: func() *Cluster {
				inputStrimziVersionsInLegacyFormat := []string{"v3", "v6", "v7"}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersionsInLegacyFormat)
				if err != nil {
					panic(err)
				}
				fmt.Println(inputStrimziVersionsInLegacyFormat)
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want: []StrimziVersion{
				StrimziVersion{Version: "v3", Ready: true},
				StrimziVersion{Version: "v6", Ready: true},
				StrimziVersion{Version: "v7", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When cluster has an empty list of available strimzi the empty list is returned",
			cluster: func() *Cluster {
				inputStrimziVersions := []StrimziVersion{}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want:    []StrimziVersion{},
			wantErr: false,
		},
		{
			name: "When cluster has a nil list of available strimzi the empty list is returned",
			cluster: func() *Cluster {
				res := Cluster{AvailableStrimziVersions: nil}
				return &res
			},
			want:    []StrimziVersion{},
			wantErr: false,
		},
		{
			name: "When cluster has an invalid JSON an error is returned",
			cluster: func() *Cluster {
				res := Cluster{AvailableStrimziVersions: []byte(`"keyone": valueone`)}
				return &res
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.cluster().GetAvailableAndReadyStrimziVersions()
			gotErr := err != nil
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				t.Errorf("wantErr: %v got: %v", tt.wantErr, err)
			}
			if !reflect.DeepEqual(res, tt.want) {
				t.Errorf("want: %v got: %v", tt.want, res)
			}
		})
	}
}

func TestSetAvailableStrimziVersions(t *testing.T) {
	tests := []struct {
		name                 string
		inputStrimziVersions []StrimziVersion
		want                 []StrimziVersion
		wantErr              bool
	}{
		{
			name: "When setting a non empty ordered list of strimzi versions that list is stored as is",
			inputStrimziVersions: []StrimziVersion{
				StrimziVersion{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: true},
				StrimziVersion{Version: "strimzi-cluster-operator-v.6.0.0-0", Ready: false},
				StrimziVersion{Version: "strimzi-cluster-operator-v.7.0.0-0", Ready: true},
			},
			want: []StrimziVersion{
				StrimziVersion{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: true},
				StrimziVersion{Version: "strimzi-cluster-operator-v.6.0.0-0", Ready: false},
				StrimziVersion{Version: "strimzi-cluster-operator-v.7.0.0-0", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When setting a non empty unordered list of strimzi versions that list is stored in semver ascending order",
			inputStrimziVersions: []StrimziVersion{
				StrimziVersion{Version: "strimzi-cluster-operator-v.5.0.0-0", Ready: true},
				StrimziVersion{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: false},
				StrimziVersion{Version: "strimzi-cluster-operator-v.2.0.0-0", Ready: true},
			},
			want: []StrimziVersion{
				StrimziVersion{Version: "strimzi-cluster-operator-v.2.0.0-0", Ready: true},
				StrimziVersion{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: false},
				StrimziVersion{Version: "strimzi-cluster-operator-v.5.0.0-0", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When setting a non empty unordered list of strimzi versions that list is stored in semver ascending order (case 2)",
			inputStrimziVersions: []StrimziVersion{
				StrimziVersion{Version: "strimzi-cluster-operator-v.5.10.0-3", Ready: true},
				StrimziVersion{Version: "strimzi-cluster-operator-v.5.8.0-9", Ready: false},
				StrimziVersion{Version: "strimzi-cluster-operator-v.2.0.0-0", Ready: true},
			},
			want: []StrimziVersion{
				StrimziVersion{Version: "strimzi-cluster-operator-v.2.0.0-0", Ready: true},
				StrimziVersion{Version: "strimzi-cluster-operator-v.5.8.0-9", Ready: false},
				StrimziVersion{Version: "strimzi-cluster-operator-v.5.10.0-3", Ready: true},
			},
			wantErr: false,
		},
		{
			name:                 "When setting an empty list of strimzi versions that list is stored as the empty list",
			inputStrimziVersions: []StrimziVersion{},
			want:                 []StrimziVersion{},
			wantErr:              false,
		},
		{
			name:                 "When setting a nil list of strimzi versions that list is stored as the empty list",
			inputStrimziVersions: nil,
			want:                 []StrimziVersion{},
			wantErr:              false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cluster := &Cluster{}
			err := cluster.SetAvailableStrimziVersions(tt.inputStrimziVersions)
			gotErr := err != nil
			errResultTestFailed := false
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				errResultTestFailed = true
				t.Errorf("wantErr: %v got: %v", tt.wantErr, gotErr)
			}

			if !errResultTestFailed {
				var got []StrimziVersion
				err := json.Unmarshal(cluster.AvailableStrimziVersions, &got)
				if err != nil {
					panic(err)
				}

				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("want: %v got: %v", tt.want, got)
				}
			}
		})
	}
}

func TestCompare(t *testing.T) {
	tests := []struct {
		name                 string
		inputStrimziVersion1 StrimziVersion
		inputStrimziVersion2 StrimziVersion
		want                 int
		wantErr              bool
	}{
		{
			name:                 "When inputStrimziVersion1 is smaller than inputStrimziVersion2 -1 is returned",
			inputStrimziVersion1: StrimziVersion{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: true},
			inputStrimziVersion2: StrimziVersion{Version: "strimzi-cluster-operator-v.6.0.0-0", Ready: false},
			want:                 -1,
			wantErr:              false,
		},
		{
			name:                 "When inputStrimziVersion1 is equal than inputStrimziVersion2 0 is returned",
			inputStrimziVersion1: StrimziVersion{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: true},
			inputStrimziVersion2: StrimziVersion{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: false},
			want:                 0,
			wantErr:              false,
		},
		{
			name:                 "When inputStrimziVersion1 is bigger than inputStrimziVersion2 1 is returned",
			inputStrimziVersion1: StrimziVersion{Version: "strimzi-cluster-operator-v.6.0.0-0", Ready: true},
			inputStrimziVersion2: StrimziVersion{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: false},
			want:                 1,
			wantErr:              false,
		},
		{
			name:                 "Check that semver-level comparison is performed",
			inputStrimziVersion1: StrimziVersion{Version: "strimzi-cluster-operator-v.6.3.10-6", Ready: true},
			inputStrimziVersion2: StrimziVersion{Version: "strimzi-cluster-operator-v.6.3.8-9", Ready: false},
			want:                 1,
			wantErr:              false,
		},
		{
			name:                 "When inputStrimziVersion1 is empty an error is returned",
			inputStrimziVersion1: StrimziVersion{Version: "", Ready: true},
			inputStrimziVersion2: StrimziVersion{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: false},
			wantErr:              true,
		},
		{
			name:                 "When inputStrimziVersion2 is empty an error is returned",
			inputStrimziVersion1: StrimziVersion{Version: "strimzi-cluster-operator-v.6.0.0-0", Ready: true},
			inputStrimziVersion2: StrimziVersion{Version: "", Ready: false},
			wantErr:              true,
		},
		{
			name:                 "When inputStrimziVersion1 has an invalid semver version format an error is returned",
			inputStrimziVersion1: StrimziVersion{Version: "strimzi-cluster-operator-v.6invalid.0.0-0", Ready: true},
			inputStrimziVersion2: StrimziVersion{Version: "strimzi-cluster-operator-v.7.0.0-0", Ready: false},
			wantErr:              true,
		},
		{
			name:                 "When inputStrimziVersion1 has an invalid expected format an error is returned",
			inputStrimziVersion1: StrimziVersion{Version: "strimzi-cluster-operator-v.6.0.0", Ready: true},
			inputStrimziVersion2: StrimziVersion{Version: "strimzi-cluster-operator-v.7.0.0-0", Ready: false},
			wantErr:              true,
		},
		{
			name:                 "When inputStrimziVersion2 has an invalid semver version format an error is returned",
			inputStrimziVersion1: StrimziVersion{Version: "strimzi-cluster-operator-v.6.0.0-0", Ready: true},
			inputStrimziVersion2: StrimziVersion{Version: "strimzi-cluster-operator-v.6invalid.0.0-0", Ready: false},
			wantErr:              true,
		},
		{
			name:                 "When inputStrimziVersion2 has an invalid expected format an error is returned",
			inputStrimziVersion1: StrimziVersion{Version: "strimzi-cluster-operator-v.7.0.0-0", Ready: true},
			inputStrimziVersion2: StrimziVersion{Version: "strimzi-cluster-operator-v.6.0.0", Ready: true},
			wantErr:              true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.inputStrimziVersion1.Compare(tt.inputStrimziVersion2)
			gotErr := err != nil
			errResultTestFailed := false
			if !reflect.DeepEqual(gotErr, tt.wantErr) {
				errResultTestFailed = true
				t.Errorf("wantErr: %v got: %v", tt.wantErr, gotErr)
			}

			if !errResultTestFailed {
				if !reflect.DeepEqual(got, tt.want) {
					t.Errorf("want: %v got: %v", tt.want, got)
				}
			}
		})
	}
}
