package config

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/profiles"
	"github.com/onsi/gomega"
	"io/ioutil"
	"testing"
)

const quotaConfigFileOk = `
---
- profile-name: default-profile
- profile-name: evaluation-profile
  quotas:
    namespace-quota:
      connectors: 4
      memory-requests: "1Gi"
      memory-limits: "2Gi"
      cpu-requests: "1"
      cpu-limits: "2"

`

const quotaConfigFileNoDefault = `
---
- profile-name: evaluation-profile
  quotas:
    namespace-quota:
      connectors: 4
      memory-requests: "1Gi"
      memory-limits: "2Gi"
      cpu-requests: "1"
      cpu-limits: "2"

`

const quotaConfigFileNoEval = `
---
- profile-name: default-profile
`

func TestConnectorsQuotaConfig_ReadFiles(t *testing.T) {
	tests := []struct {
		name   string
		config ConnectorsQuotaConfig
		err    string
	}{
		{
			name: "quotaConfigFileOk",
			config: ConnectorsQuotaConfig{
				connectorsQuotaMap:           make(ConnectorsQuotaProfileMap),
				ConnectorsQuotaConfigFile:    createFile(t, []byte(quotaConfigFileOk)),
				EvalNamespaceQuotaProfile:    profiles.EvaluationProfileName,
				DefaultNamespaceQuotaProfile: profiles.DefaultProfileName,
			},
			err: "",
		},
		{
			name: "quotaConfigFileNoDefault",
			config: ConnectorsQuotaConfig{
				connectorsQuotaMap:           make(ConnectorsQuotaProfileMap),
				ConnectorsQuotaConfigFile:    createFile(t, []byte(quotaConfigFileNoDefault)),
				EvalNamespaceQuotaProfile:    profiles.EvaluationProfileName,
				DefaultNamespaceQuotaProfile: profiles.DefaultProfileName,
			},
			err: "is missing default namespace quota profile",
		},
		{
			name: "quotaConfigFileNoEval",
			config: ConnectorsQuotaConfig{
				connectorsQuotaMap:           make(ConnectorsQuotaProfileMap),
				ConnectorsQuotaConfigFile:    createFile(t, []byte(quotaConfigFileNoEval)),
				EvalNamespaceQuotaProfile:    profiles.EvaluationProfileName,
				DefaultNamespaceQuotaProfile: profiles.DefaultProfileName,
			},
			err: "is missing evaluation namespace quota profile",
		},
	}

	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			c := tt.config

			err := c.ReadFiles()
			if err != nil {
				if tt.err != "" {
					g.Expect(err.Error()).To(gomega.MatchRegexp(tt.err))
				} else {
					t.Error(err)
				}
			} else if tt.err != "" {
				t.Errorf("ReadFiles() suceed but wantErr %v", tt.err)
			}
		})
	}
}

func createFile(t *testing.T, content []byte) string {
	file, err := ioutil.TempFile(t.TempDir(), t.Name())
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		if err := file.Close(); err != nil {
			t.Fatal(err)
		}
	}()

	_, err = file.Write(content)
	if err != nil {
		t.Fatal(err)
	}

	return file.Name()
}
