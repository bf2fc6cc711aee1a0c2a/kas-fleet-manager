package api

import (
	"encoding/json"
	"testing"

	. "github.com/onsi/gomega"
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.cluster().GetAvailableStrimziVersions()
			gotErr := err != nil
			if gotErr != tt.wantErr {
				t.Errorf("wantErr: %v got: %v", tt.wantErr, err)
			}
			Expect(res).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			res, err := tt.cluster().GetAvailableAndReadyStrimziVersions()
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(res).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
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

			Expect(got).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.inputStrimziVersion1.Compare(tt.inputStrimziVersion2)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(got).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			got, err := StrimziVersionsDeepSort(tt.args.versions)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(got).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareBuildAwareSemanticVersions(tt.version1, tt.version2)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(got).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			got, err := CompareSemanticVersionsMajorAndMinor(tt.current, tt.desired)
			Expect(err != nil).To(Equal(tt.wantErr))
			Expect(got).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			index := tt.cluster.Index()
			Expect(index).To(Equal(tt.want))
		})
	}
}

func Test_ClusterTypes_BeforeCreate(t *testing.T) {
	RegisterTestingT(t)

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
		Expect(clusterWithFieldsSet.ID).To(Equal(id))
		Expect(clusterWithFieldsSet.Status).To(Equal(status))
		Expect(clusterWithFieldsSet.SupportedInstanceType).To(Equal(supportedInstanceType))
		Expect(err).ToNot(HaveOccurred())
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
		Expect(err).ToNot(HaveOccurred())
		Expect(clusterWithEmptyValue.ID).ToNot(BeEmpty())
		Expect(clusterWithEmptyValue.Status).To(Equal(ClusterAccepted))
		Expect(clusterWithEmptyValue.SupportedInstanceType).To(Equal(AllInstanceTypeSupport.String()))
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			res := tt.k.CompareTo(tt.k1)
			Expect(res).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			res := tt.providerType.String()
			Expect(res).To(Equal(tt.want))
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

	RegisterTestingT(t)

	for _, testcase := range tests {
		tt := testcase
		t.Run(tt.name, func(t *testing.T) {
			res := tt.status.String()
			Expect(res).To(Equal(tt.want))
		})
	}
}
