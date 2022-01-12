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
		{
			name: "Dinosaur and Dinosaur IBP versions are stored and in sorted order",
			inputStrimziVersions: []StrimziVersion{
				StrimziVersion{
					Version: "strimzi-cluster-operator-v.5.10.0-3",
					Ready:   true,
					DinosaurVersions: []DinosaurVersion{
						DinosaurVersion{Version: "2.7.5"},
						DinosaurVersion{Version: "2.7.3"},
					},
					DinosaurIBPVersions: []DinosaurIBPVersion{
						DinosaurIBPVersion{Version: "2.8"},
						DinosaurIBPVersion{Version: "2.7"},
					},
				},
				StrimziVersion{
					Version: "strimzi-cluster-operator-v.5.8.0-9",
					Ready:   false,
					DinosaurVersions: []DinosaurVersion{
						DinosaurVersion{Version: "2.9.4"},
						DinosaurVersion{Version: "2.2.1"},
					},
					DinosaurIBPVersions: []DinosaurIBPVersion{
						DinosaurIBPVersion{Version: "2.5"},
						DinosaurIBPVersion{Version: "2.6"},
					},
				},
				StrimziVersion{
					Version: "strimzi-cluster-operator-v.2.0.0-0",
					Ready:   true,
					DinosaurVersions: []DinosaurVersion{
						DinosaurVersion{Version: "4.5.6"},
						DinosaurVersion{Version: "1.2.3"},
					},
					DinosaurIBPVersions: []DinosaurIBPVersion{
						DinosaurIBPVersion{Version: "2.3"},
						DinosaurIBPVersion{Version: "2.2"},
					},
				},
			},
			want: []StrimziVersion{
				StrimziVersion{
					Version: "strimzi-cluster-operator-v.2.0.0-0",
					Ready:   true,
					DinosaurVersions: []DinosaurVersion{
						DinosaurVersion{Version: "1.2.3"},
						DinosaurVersion{Version: "4.5.6"},
					},
					DinosaurIBPVersions: []DinosaurIBPVersion{
						DinosaurIBPVersion{Version: "2.2"},
						DinosaurIBPVersion{Version: "2.3"},
					},
				},
				StrimziVersion{
					Version: "strimzi-cluster-operator-v.5.8.0-9",
					Ready:   false,
					DinosaurVersions: []DinosaurVersion{
						DinosaurVersion{Version: "2.2.1"},
						DinosaurVersion{Version: "2.9.4"},
					},
					DinosaurIBPVersions: []DinosaurIBPVersion{
						DinosaurIBPVersion{Version: "2.5"},
						DinosaurIBPVersion{Version: "2.6"},
					},
				},
				StrimziVersion{
					Version: "strimzi-cluster-operator-v.5.10.0-3",
					Ready:   true,
					DinosaurVersions: []DinosaurVersion{
						DinosaurVersion{Version: "2.7.3"},
						DinosaurVersion{Version: "2.7.5"},
					},
					DinosaurIBPVersions: []DinosaurIBPVersion{
						DinosaurIBPVersion{Version: "2.7"},
						DinosaurIBPVersion{Version: "2.8"},
					},
				},
			},
			wantErr: false,
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

func Test_StrimziVersionsDeepSort(t *testing.T) {
	type args struct {
		versions []StrimziVersion
	}

	tests := []struct {
		name    string
		args    args
		cluster func() *Cluster
		want    []StrimziVersion
		wantErr bool
	}{
		{
			name: "When versions to sort is empty result is empty",
			args: args{
				versions: []StrimziVersion{},
			},
			want: []StrimziVersion{},
		},
		{
			name: "When versions to sort is nil result is nil",
			args: args{
				versions: nil,
			},
			want: nil,
		},
		{
			name: "When one of the strimzi versions does not follow semver an error is returned",
			args: args{
				[]StrimziVersion{StrimziVersion{Version: "strimzi-cluster-operator-v.nonsemver243-0"}, StrimziVersion{Version: "strimzi-cluster-operator-v.2.5.6-0"}},
			},
			wantErr: true,
		},
		{
			name: "All different versions are deeply sorted",
			args: args{
				versions: []StrimziVersion{
					StrimziVersion{
						Version: "strimzi-cluster-operator-v.2.7.5-0",
						DinosaurVersions: []DinosaurVersion{
							DinosaurVersion{Version: "1.5.8"},
							DinosaurVersion{Version: "0.7.1"},
							DinosaurVersion{Version: "1.5.1"},
						},
						DinosaurIBPVersions: []DinosaurIBPVersion{
							DinosaurIBPVersion{Version: "2.8"},
							DinosaurIBPVersion{Version: "1.2"},
							DinosaurIBPVersion{Version: "2.4"},
						},
					},
					StrimziVersion{
						Version: "strimzi-cluster-operator-v.2.7.3-0",
						DinosaurVersions: []DinosaurVersion{
							DinosaurVersion{Version: "1.0.0"},
							DinosaurVersion{Version: "2.0.0"},
							DinosaurVersion{Version: "5.0.0"},
						},
						DinosaurIBPVersions: []DinosaurIBPVersion{
							DinosaurIBPVersion{Version: "4.0"},
							DinosaurIBPVersion{Version: "2.0"},
							DinosaurIBPVersion{Version: "3.5"},
						},
					},
					StrimziVersion{
						Version: "strimzi-cluster-operator-v.2.5.2-0",
						DinosaurVersions: []DinosaurVersion{
							DinosaurVersion{Version: "2.6.1"},
							DinosaurVersion{Version: "5.7.2"},
							DinosaurVersion{Version: "2.3.5"},
						},
						DinosaurIBPVersions: []DinosaurIBPVersion{
							DinosaurIBPVersion{Version: "1.2"},
							DinosaurIBPVersion{Version: "1.1"},
							DinosaurIBPVersion{Version: "5.1"},
						},
					},
				},
			},
			want: []StrimziVersion{
				StrimziVersion{
					Version: "strimzi-cluster-operator-v.2.5.2-0",
					DinosaurVersions: []DinosaurVersion{
						DinosaurVersion{Version: "2.3.5"},
						DinosaurVersion{Version: "2.6.1"},
						DinosaurVersion{Version: "5.7.2"},
					},
					DinosaurIBPVersions: []DinosaurIBPVersion{
						DinosaurIBPVersion{Version: "1.1"},
						DinosaurIBPVersion{Version: "1.2"},
						DinosaurIBPVersion{Version: "5.1"},
					},
				},
				StrimziVersion{
					Version: "strimzi-cluster-operator-v.2.7.3-0",
					DinosaurVersions: []DinosaurVersion{
						DinosaurVersion{Version: "1.0.0"},
						DinosaurVersion{Version: "2.0.0"},
						DinosaurVersion{Version: "5.0.0"},
					},
					DinosaurIBPVersions: []DinosaurIBPVersion{
						DinosaurIBPVersion{Version: "2.0"},
						DinosaurIBPVersion{Version: "3.5"},
						DinosaurIBPVersion{Version: "4.0"},
					},
				},
				StrimziVersion{
					Version: "strimzi-cluster-operator-v.2.7.5-0",
					DinosaurVersions: []DinosaurVersion{
						DinosaurVersion{Version: "0.7.1"},
						DinosaurVersion{Version: "1.5.1"},
						DinosaurVersion{Version: "1.5.8"},
					},
					DinosaurIBPVersions: []DinosaurIBPVersion{
						DinosaurIBPVersion{Version: "1.2"},
						DinosaurIBPVersion{Version: "2.4"},
						DinosaurIBPVersion{Version: "2.8"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := StrimziVersionsDeepSort(tt.args.versions)
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

func TestCompareSemanticVersionsMajorAndMinor(t *testing.T) {
	tests := []struct {
		name    string
		current string
		desired string
		want    int
		wantErr bool
	}{
		{
			name:    "When desired major is smaller than current major, 1 is returned",
			current: "3.6.0",
			desired: "2.6.0",
			want:    1,
			wantErr: false,
		},
		{
			name:    "When desired major is greater than current major -1, is returned",
			current: "2.7.0",
			desired: "3.7.0",
			want:    -1,
			wantErr: false,
		},
		{
			name:    "When major versions are equal and desired minor is greater than current minor, -1 is returned",
			current: "2.7.0",
			desired: "2.8.0",
			want:    -1,
			wantErr: false,
		},
		{
			name:    "When major versions are equal and desired minor is smaller than current minor, 1 is returned",
			current: "2.8.0",
			desired: "2.7.0",
			want:    1,
			wantErr: false,
		},
		{
			name:    "When major versions are equal and desired minor is equal to current minor, 0 is returned",
			current: "2.7.0",
			desired: "2.7.0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "When major and minor versions are equal and desired patch is equal to current patch, 0 is returned",
			current: "2.7.0",
			desired: "2.7.0",
			want:    0,
			wantErr: false,
		},
		{
			name:    "When major and minor versions are equal and desired patch is greater than current patch, 0 is returned",
			current: "2.7.0",
			desired: "2.7.1",
			want:    0,
			wantErr: false,
		},
		{
			name:    "When major and minor versions are equal and desired patch is smaller than current patch, 0 is returned",
			current: "2.7.2",
			desired: "2.7.1",
			want:    0,
			wantErr: false,
		},
		{
			name:    "When current is empty an error is returned",
			current: "",
			desired: "2.7.1",
			wantErr: true,
		},
		{
			name:    "When desired is empty an error is returned",
			current: "2.7.1",
			desired: "",
			wantErr: true,
		},
		{
			name:    "When current has an invalid semver version format an error is returned",
			current: "2invalid.6.0",
			desired: "2.7.1",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareSemanticVersionsMajorAndMinor(tt.current, tt.desired)
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
