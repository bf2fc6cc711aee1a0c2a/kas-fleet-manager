package api

import (
	"encoding/json"
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
					StrimziVersion("v3"),
					StrimziVersion("v6"),
					StrimziVersion("v7"),
				}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want: []StrimziVersion{
				StrimziVersion("v3"),
				StrimziVersion("v6"),
				StrimziVersion("v7"),
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
				t.Errorf("want: %v got: %v", tt.wantErr, err)
			}
			if !reflect.DeepEqual(res, tt.want) {
				t.Errorf("wantErr: %v got: %v", tt.want, res)
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
				StrimziVersion("v3"),
				StrimziVersion("v6"),
				StrimziVersion("v7"),
			},
			want: []StrimziVersion{
				StrimziVersion("v3"),
				StrimziVersion("v6"),
				StrimziVersion("v7"),
			},
			wantErr: false,
		},
		{
			name: "When setting a non empty unordered list of strimzi versions that list is stored in a lexicographically ascending order",
			inputStrimziVersions: []StrimziVersion{
				StrimziVersion("v5"),
				StrimziVersion("v3"),
				StrimziVersion("v2"),
			},
			want: []StrimziVersion{
				StrimziVersion("v2"),
				StrimziVersion("v3"),
				StrimziVersion("v5"),
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
