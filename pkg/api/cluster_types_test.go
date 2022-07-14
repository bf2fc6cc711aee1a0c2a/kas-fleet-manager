package api

import (
	"encoding/json"
	"testing"

	"github.com/onsi/gomega"
)

func Test_Cluster_GetAvailableStrimziVersions(t *testing.T) {
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
					{Version: "v3", Ready: true},
					{Version: "v6", Ready: false},
					{Version: "v7", Ready: true},
				}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want: []StrimziVersion{
				{Version: "v3", Ready: true},
				{Version: "v6", Ready: false},
				{Version: "v7", Ready: true},
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res, err := tt.cluster().GetAvailableStrimziVersions()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr), "wantErr: %v got: %v", tt.wantErr, err)
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Cluster_GetLatestAvailableStrimziVersion(t *testing.T) {
	tests := []struct {
		name    string
		cluster func() *Cluster
		want    *StrimziVersion
		wantErr bool
	}{
		{
			name: "When cluster has a non empty list of available strimzi versions the latest one is returned",
			cluster: func() *Cluster {
				inputStrimziVersions := []StrimziVersion{
					{Version: "v3", Ready: true},
					{Version: "v6", Ready: false},
					{Version: "v7", Ready: false},
				}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want:    &StrimziVersion{Version: "v7", Ready: false},
			wantErr: false,
		},
		{
			name: "When cluster has an empty list of available strimzi nil is returned",
			cluster: func() *Cluster {
				inputStrimziVersions := []StrimziVersion{}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "When cluster has a nil list of available strimzi the empty list is returned",
			cluster: func() *Cluster {
				res := Cluster{AvailableStrimziVersions: nil}
				return &res
			},
			want:    nil,
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			res, err := tt.cluster().GetLatestAvailableStrimziVersion()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr), "wantErr: %v got: %v", tt.wantErr, err)
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Cluster_GetAvailableAndReadyStrimziVersions(t *testing.T) {
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
					{Version: "v3", Ready: true},
					{Version: "v6", Ready: false},
					{Version: "v7", Ready: true},
				}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want: []StrimziVersion{
				{Version: "v3", Ready: true},
				{Version: "v7", Ready: true},
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res, err := tt.cluster().GetAvailableAndReadyStrimziVersions()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Cluster_GetLatestAvailableAndReadyStrimziVersion(t *testing.T) {
	tests := []struct {
		name    string
		cluster func() *Cluster
		want    *StrimziVersion
		wantErr bool
	}{
		{
			name: "When cluster has a non empty list of available strimzi versions those ready returned",
			cluster: func() *Cluster {
				inputStrimziVersions := []StrimziVersion{
					{Version: "v3", Ready: true},
					{Version: "v7", Ready: true},
					{Version: "v9", Ready: false},
				}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want:    &StrimziVersion{Version: "v7", Ready: true},
			wantErr: false,
		},
		{
			name: "When cluster has an empty list of available strimzi nil is returned",
			cluster: func() *Cluster {
				inputStrimziVersions := []StrimziVersion{}
				inputStrimziVersionsJSON, err := json.Marshal(inputStrimziVersions)
				if err != nil {
					panic(err)
				}
				res := Cluster{AvailableStrimziVersions: inputStrimziVersionsJSON}
				return &res
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "When cluster has a nil list of available strimzi nil is returned",
			cluster: func() *Cluster {
				res := Cluster{AvailableStrimziVersions: nil}
				return &res
			},
			want:    nil,
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			g := gomega.NewWithT(t)
			res, err := tt.cluster().GetLatestAvailableAndReadyStrimziVersion()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(res).To(gomega.Equal(tt.want))
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
				{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: true},
				{Version: "strimzi-cluster-operator-v.6.0.0-0", Ready: false},
				{Version: "strimzi-cluster-operator-v.7.0.0-0", Ready: true},
			},
			want: []StrimziVersion{
				{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: true},
				{Version: "strimzi-cluster-operator-v.6.0.0-0", Ready: false},
				{Version: "strimzi-cluster-operator-v.7.0.0-0", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When setting a non empty unordered list of strimzi versions that list is stored in semver ascending order",
			inputStrimziVersions: []StrimziVersion{
				{Version: "strimzi-cluster-operator-v.5.0.0-0", Ready: true},
				{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: false},
				{Version: "strimzi-cluster-operator-v.2.0.0-0", Ready: true},
			},
			want: []StrimziVersion{
				{Version: "strimzi-cluster-operator-v.2.0.0-0", Ready: true},
				{Version: "strimzi-cluster-operator-v.3.0.0-0", Ready: false},
				{Version: "strimzi-cluster-operator-v.5.0.0-0", Ready: true},
			},
			wantErr: false,
		},
		{
			name: "When setting a non empty unordered list of strimzi versions that list is stored in semver ascending order (case 2)",
			inputStrimziVersions: []StrimziVersion{
				{Version: "strimzi-cluster-operator-v.5.10.0-3", Ready: true},
				{Version: "strimzi-cluster-operator-v.5.8.0-9", Ready: false},
				{Version: "strimzi-cluster-operator-v.2.0.0-0", Ready: true},
			},
			want: []StrimziVersion{
				{Version: "strimzi-cluster-operator-v.2.0.0-0", Ready: true},
				{Version: "strimzi-cluster-operator-v.5.8.0-9", Ready: false},
				{Version: "strimzi-cluster-operator-v.5.10.0-3", Ready: true},
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
			name: "Kafka and Kafka IBP versions are stored and in sorted order",
			inputStrimziVersions: []StrimziVersion{
				{
					Version: "strimzi-cluster-operator-v.5.10.0-3",
					Ready:   true,
					KafkaVersions: []KafkaVersion{
						{Version: "2.7.5"},
						{Version: "2.7.3"},
					},
					KafkaIBPVersions: []KafkaIBPVersion{
						{Version: "2.8"},
						{Version: "2.7"},
					},
				},
				{
					Version: "strimzi-cluster-operator-v.5.8.0-9",
					Ready:   false,
					KafkaVersions: []KafkaVersion{
						{Version: "2.9.4"},
						{Version: "2.2.1"},
					},
					KafkaIBPVersions: []KafkaIBPVersion{
						{Version: "2.5"},
						{Version: "2.6"},
					},
				},
				{
					Version: "strimzi-cluster-operator-v.2.0.0-0",
					Ready:   true,
					KafkaVersions: []KafkaVersion{
						{Version: "4.5.6"},
						{Version: "1.2.3"},
					},
					KafkaIBPVersions: []KafkaIBPVersion{
						{Version: "2.3"},
						{Version: "2.2"},
					},
				},
			},
			want: []StrimziVersion{
				{
					Version: "strimzi-cluster-operator-v.2.0.0-0",
					Ready:   true,
					KafkaVersions: []KafkaVersion{
						{Version: "1.2.3"},
						{Version: "4.5.6"},
					},
					KafkaIBPVersions: []KafkaIBPVersion{
						{Version: "2.2"},
						{Version: "2.3"},
					},
				},
				{
					Version: "strimzi-cluster-operator-v.5.8.0-9",
					Ready:   false,
					KafkaVersions: []KafkaVersion{
						{Version: "2.2.1"},
						{Version: "2.9.4"},
					},
					KafkaIBPVersions: []KafkaIBPVersion{
						{Version: "2.5"},
						{Version: "2.6"},
					},
				},
				{
					Version: "strimzi-cluster-operator-v.5.10.0-3",
					Ready:   true,
					KafkaVersions: []KafkaVersion{
						{Version: "2.7.3"},
						{Version: "2.7.5"},
					},
					KafkaIBPVersions: []KafkaIBPVersion{
						{Version: "2.7"},
						{Version: "2.8"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			cluster := &Cluster{}
			err := cluster.SetAvailableStrimziVersions(tt.inputStrimziVersions)
			gotErr := err != nil

			if gotErr != tt.wantErr {
				t.Errorf("wantErr: %v got: %v", tt.wantErr, gotErr)
				return
			}

			var got []StrimziVersion
			err = json.Unmarshal(cluster.AvailableStrimziVersions, &got)
			if err != nil {
				panic(err)
			}

			g.Expect(got).To(gomega.Equal(tt.want))
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

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got, err := tt.inputStrimziVersion1.Compare(tt.inputStrimziVersion2)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
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
				[]StrimziVersion{{Version: "strimzi-cluster-operator-v.nonsemver243-0"}, {Version: "strimzi-cluster-operator-v.2.5.6-0"}},
			},
			wantErr: true,
		},
		{
			name: "All different versions are deeply sorted",
			args: args{
				versions: []StrimziVersion{
					{
						Version: "strimzi-cluster-operator-v.2.7.5-0",
						KafkaVersions: []KafkaVersion{
							{Version: "1.5.8"},
							{Version: "0.7.1"},
							{Version: "1.5.1"},
						},
						KafkaIBPVersions: []KafkaIBPVersion{
							{Version: "2.8"},
							{Version: "1.2"},
							{Version: "2.4"},
						},
					},
					{
						Version: "strimzi-cluster-operator-v.2.7.3-0",
						KafkaVersions: []KafkaVersion{
							{Version: "1.0.0"},
							{Version: "2.0.0"},
							{Version: "5.0.0"},
						},
						KafkaIBPVersions: []KafkaIBPVersion{
							{Version: "4.0"},
							{Version: "2.0"},
							{Version: "3.5"},
						},
					},
					{
						Version: "strimzi-cluster-operator-v.2.5.2-0",
						KafkaVersions: []KafkaVersion{
							{Version: "2.6.1"},
							{Version: "5.7.2"},
							{Version: "2.3.5"},
						},
						KafkaIBPVersions: []KafkaIBPVersion{
							{Version: "1.2"},
							{Version: "1.1"},
							{Version: "5.1"},
						},
					},
				},
			},
			want: []StrimziVersion{
				{
					Version: "strimzi-cluster-operator-v.2.5.2-0",
					KafkaVersions: []KafkaVersion{
						{Version: "2.3.5"},
						{Version: "2.6.1"},
						{Version: "5.7.2"},
					},
					KafkaIBPVersions: []KafkaIBPVersion{
						{Version: "1.1"},
						{Version: "1.2"},
						{Version: "5.1"},
					},
				},
				{
					Version: "strimzi-cluster-operator-v.2.7.3-0",
					KafkaVersions: []KafkaVersion{
						{Version: "1.0.0"},
						{Version: "2.0.0"},
						{Version: "5.0.0"},
					},
					KafkaIBPVersions: []KafkaIBPVersion{
						{Version: "2.0"},
						{Version: "3.5"},
						{Version: "4.0"},
					},
				},
				{
					Version: "strimzi-cluster-operator-v.2.7.5-0",
					KafkaVersions: []KafkaVersion{
						{Version: "0.7.1"},
						{Version: "1.5.1"},
						{Version: "1.5.8"},
					},
					KafkaIBPVersions: []KafkaIBPVersion{
						{Version: "1.2"},
						{Version: "2.4"},
						{Version: "2.8"},
					},
				},
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got, err := StrimziVersionsDeepSort(tt.args.versions)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_CompareBuildAwareSemanticVersions(t *testing.T) {
	tests := []struct {
		name     string
		version1 string
		version2 string
		want     int
		wantErr  bool
	}{
		{
			name:     "when v1 is greater than v2 1 is returned",
			version1: "1.0.1",
			version2: "1.0.0",
			want:     1,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got, err := CompareBuildAwareSemanticVersions(tt.version1, tt.version2)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_CompareSemanticVersionsMajorAndMinor(t *testing.T) {
	tests := []struct {
		name    string
		current string
		desired string
		want    int
		wantErr bool
	}{
		{
			name:    "when major and minor version are equal 0 is returned",
			current: "2.7.0",
			desired: "2.7.0",
			want:    0,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			got, err := CompareSemanticVersionsMajorAndMinor(tt.current, tt.desired)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			g.Expect(got).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Cluster_GetLatestKafkaVersion(t *testing.T) {
	tests := []struct {
		name                  string
		strimziVersionFactory func() *StrimziVersion
		want                  *KafkaVersion
	}{
		{
			name: "returns the latest element in the kafka versions list",
			strimziVersionFactory: func() *StrimziVersion {
				strimziVersion := &StrimziVersion{
					KafkaVersions: []KafkaVersion{
						KafkaVersion{Version: "1.5.3"},
						KafkaVersion{Version: "2.3.6"},
						KafkaVersion{Version: "1.2.0"},
						KafkaVersion{Version: "0.6.2"},
					},
				}
				versions := []StrimziVersion{*strimziVersion}
				sortedVersions, err := StrimziVersionsDeepSort(versions)
				if err != nil || len(sortedVersions) != 1 {
					panic("unexpected test error")
				}
				return &sortedVersions[0]
			},
			want: &KafkaVersion{Version: "2.3.6"},
		},
		{
			name: "returns an nil if there are no kafka versions in the strimzi version",
			strimziVersionFactory: func() *StrimziVersion {
				return &StrimziVersion{}
			},
			want: nil,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(test *testing.T) {
			test.Parallel()
			g := gomega.NewWithT(t)
			strimziVersion := tt.strimziVersionFactory()
			res := strimziVersion.GetLatestKafkaVersion()
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Cluster_GetLatestKafkaIBPVersion(t *testing.T) {
	tests := []struct {
		name                  string
		strimziVersionFactory func() *StrimziVersion
		want                  *KafkaIBPVersion
	}{
		{
			name: "returns the latest element in the kafka versions list",
			strimziVersionFactory: func() *StrimziVersion {
				strimziVersion := &StrimziVersion{
					KafkaIBPVersions: []KafkaIBPVersion{
						KafkaIBPVersion{Version: "1.5.3"},
						KafkaIBPVersion{Version: "2.3.6"},
						KafkaIBPVersion{Version: "1.2.0"},
						KafkaIBPVersion{Version: "0.6.2"},
					},
				}
				versions := []StrimziVersion{*strimziVersion}
				sortedVersions, err := StrimziVersionsDeepSort(versions)
				if err != nil || len(sortedVersions) != 1 {
					panic("unexpected test error")
				}
				return &sortedVersions[0]
			},
			want: &KafkaIBPVersion{Version: "2.3.6"},
		},
		{
			name: "returns an nil if there are no kafka versions in the strimzi version",
			strimziVersionFactory: func() *StrimziVersion {
				return &StrimziVersion{}
			},
			want: nil,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(test *testing.T) {
			test.Parallel()
			g := gomega.NewWithT(t)
			strimziVersion := tt.strimziVersionFactory()
			res := strimziVersion.GetLatestKafkaIBPVersion()
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ClusterTypes_Index(t *testing.T) {
	cluster := &Cluster{
		Meta: Meta{
			ID: "test-id",
		},
		CloudProvider:            "",
		ClusterID:                "",
		ExternalID:               "",
		MultiAZ:                  true,
		Region:                   "",
		Status:                   "accepted",
		StatusDetails:            "",
		IdentityProviderID:       "",
		ClusterDNS:               "",
		ClientID:                 "",
		ClientSecret:             "",
		ProviderType:             "",
		ProviderSpec:             JSON{},
		ClusterSpec:              JSON{},
		AvailableStrimziVersions: JSON{},
		SupportedInstanceType:    "",
	}

	tests := []struct {
		name    string
		cluster ClusterList
		want    ClusterIndex
	}{
		{
			name:    "returns an empty index map when leader lease list is empty",
			cluster: ClusterList{},
			want:    ClusterIndex{},
		},
		{
			name: "returns an index where a cluster id point to cluster represented by this ID",
			cluster: ClusterList{
				cluster,
			},
			want: ClusterIndex{
				cluster.ID: cluster,
			},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			index := tt.cluster.Index()
			g.Expect(index).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ClusterTypes_BeforeCreate(t *testing.T) {
	g := gomega.NewWithT(t)
	id := "some-id"
	status := ClusterCleanup
	supportedInstanceType := "standard"
	clusterWithFieldsSet := &Cluster{
		Status:                status,
		SupportedInstanceType: supportedInstanceType,
		Meta: Meta{
			ID: id,
		},
	}

	t.Run("do not modify the values if they are set", func(t *testing.T) {
		err := clusterWithFieldsSet.BeforeCreate(nil)
		g.Expect(clusterWithFieldsSet.ID).To(gomega.Equal(id))
		g.Expect(clusterWithFieldsSet.Status).To(gomega.Equal(status))
		g.Expect(clusterWithFieldsSet.SupportedInstanceType).To(gomega.Equal(supportedInstanceType))
		g.Expect(err).ToNot(gomega.HaveOccurred())
	})

	clusterWithEmptyValue := &Cluster{
		SupportedInstanceType: "",
		Status:                "",
		Meta: Meta{
			ID: "",
		},
	}

	t.Run("set the default values if empty", func(t *testing.T) {
		err := clusterWithEmptyValue.BeforeCreate(nil)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(clusterWithEmptyValue.ID).ToNot(gomega.BeEmpty())
		g.Expect(clusterWithEmptyValue.Status).To(gomega.Equal(ClusterAccepted))
		g.Expect(clusterWithEmptyValue.SupportedInstanceType).To(gomega.Equal(AllInstanceTypeSupport.String()))
	})
}

func Test_CompareTo(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		k1      ClusterStatus
		k       ClusterStatus
		want    int
	}{
		{
			name: "cluster has a non empty list of available strimzi versions those are returned",
			k:    ClusterAccepted,
			k1:   ClusterAccepted,
			want: 0,
		},
		{
			name: " cluster has a non empty list of available strimzi versions those are returned",
			k:    ClusterReady,
			k1:   ClusterAccepted,
			want: 1,
		},
		{
			name: " cluster has a non empty list of available strimzi versions those are returned",
			k:    ClusterAccepted,
			k1:   ClusterReady,
			want: -1,
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res := tt.k.CompareTo(tt.k1)
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ClusterProviderType_String(t *testing.T) {
	tests := []struct {
		name         string
		providerType ClusterProviderType
		want         string
	}{
		{
			name:         "returns cluster provider type string",
			providerType: ClusterProviderOCM,
			want:         "ocm",
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res := tt.providerType.String()
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_ClusterStatus_String(t *testing.T) {
	tests := []struct {
		name   string
		status ClusterStatus
		want   string
	}{
		{
			name:   "return cluster status string",
			status: ClusterAccepted,
			want:   "cluster_accepted",
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			res := tt.status.String()
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Cluster_SetDynamicCapacityInfo(t *testing.T) {
	tests := []struct {
		name    string
		cluster *Cluster
		arg     map[string]DynamicCapacityInfo
		want    JSON
		wantErr bool
	}{
		{
			name:    "sets the capacity config to empty",
			arg:     map[string]DynamicCapacityInfo{},
			cluster: &Cluster{},
			wantErr: false,
			want:    JSON([]byte("{}")),
		},
		{
			name:    "sets the capacity config to null",
			arg:     nil,
			cluster: &Cluster{},
			wantErr: false,
			want:    JSON([]byte("null")),
		},
		{
			name: "sets the capacity config to the given marshelled value",
			arg: map[string]DynamicCapacityInfo{
				"key1": {
					MaxNodes:       1,
					RemainingUnits: 1,
					MaxUnits:       1,
				},
			},
			cluster: &Cluster{},
			wantErr: false,
			want:    JSON([]byte(`{"key1":{"max_nodes":1,"max_units":1,"remaining_units":1}}`)),
		},
	}

	g := gomega.NewWithT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(test *testing.T) {
			test.Parallel()
			err := tt.cluster.SetDynamicCapacityInfo(tt.arg)
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if !tt.wantErr {
				g.Expect(tt.cluster.DynamicCapacityInfo).To(gomega.Equal(tt.want))
			}
		})
	}
}

func Test_Cluster_RetrieveDynamicCapacityInfo(t *testing.T) {
	tests := []struct {
		name    string
		cluster *Cluster
		want    map[string]DynamicCapacityInfo
		wantErr bool
	}{

		{
			name: "returns parsing error error",
			want: nil,
			cluster: &Cluster{
				DynamicCapacityInfo: JSON([]byte(`"key1":{"max_nodes"`)),
			},
			wantErr: true,
		},
		{
			name: "returns empty map when json object is empty",
			want: map[string]DynamicCapacityInfo{},
			cluster: &Cluster{
				DynamicCapacityInfo: JSON([]byte("{}")),
			},
			wantErr: false,
		},
		{
			name: "returns an empty map when json object is nil",
			want: map[string]DynamicCapacityInfo{},
			cluster: &Cluster{
				DynamicCapacityInfo: nil,
			},
			wantErr: false,
		},
		{
			name: "return the capacity config to nil when it is null",
			want: nil,
			cluster: &Cluster{
				DynamicCapacityInfo: JSON([]byte("null")),
			},
			wantErr: false,
		},
		{
			name: "retrieves the capacity config from the given marshelled value",
			want: map[string]DynamicCapacityInfo{
				"key1": {
					MaxNodes:       1,
					RemainingUnits: 1,
					MaxUnits:       1,
				},
			},
			cluster: &Cluster{
				DynamicCapacityInfo: JSON([]byte(`{"key1":{"max_nodes":1,"max_units":1,"remaining_units":1}}`)),
			},
			wantErr: false,
		},
	}

	g := gomega.NewWithT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(test *testing.T) {
			test.Parallel()
			dynamicCapacityInfo, err := tt.cluster.RetrieveDynamicCapacityInfo()
			g.Expect(err != nil).To(gomega.Equal(tt.wantErr))
			if !tt.wantErr {
				g.Expect(dynamicCapacityInfo).To(gomega.Equal(tt.want))
			}
		})
	}
}

func Test_Cluster_GetSupportedInstanceTypes(t *testing.T) {
	tests := []struct {
		name    string
		cluster *Cluster
		want    []string
	}{
		{
			name: "returns correct result when there is only a supported instance type",
			cluster: &Cluster{
				SupportedInstanceType: "exampleinstancetype",
			},
			want: []string{"exampleinstancetype"},
		},
		{
			name: "returns correct result when there are multiple supported instance types",
			cluster: &Cluster{
				SupportedInstanceType: "exampleinstancetype3,exampleinstancetype2,exampleinstancetype6",
			},
			want: []string{"exampleinstancetype3", "exampleinstancetype2", "exampleinstancetype6"},
		},
		{
			name:    "returns an empty list when there are no supported instance types",
			cluster: &Cluster{},
			want:    []string{},
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(test *testing.T) {
			g := gomega.NewWithT(t)
			res := tt.cluster.GetSupportedInstanceTypes()
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}

func Test_Cluster_GetRawSupportedInstanceTypes(t *testing.T) {
	tests := []struct {
		name    string
		cluster *Cluster
		want    string
	}{
		{
			name: "returns correct result when there is only a supported instance type",
			cluster: &Cluster{
				SupportedInstanceType: "exampleinstancetype",
			},
			want: "exampleinstancetype",
		},
		{
			name: "returns correct result when there are multiple supported instance types",
			cluster: &Cluster{
				SupportedInstanceType: "exampleinstancetype3,exampleinstancetype2,exampleinstancetype6",
			},
			want: "exampleinstancetype3,exampleinstancetype2,exampleinstancetype6",
		},
		{
			name:    "returns an empty string when there are no supported instance types",
			cluster: &Cluster{},
			want:    "",
		},
	}

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(test *testing.T) {
			g := gomega.NewWithT(t)
			res := tt.cluster.GetRawSupportedInstanceTypes()
			g.Expect(res).To(gomega.Equal(tt.want))
		})
	}
}
