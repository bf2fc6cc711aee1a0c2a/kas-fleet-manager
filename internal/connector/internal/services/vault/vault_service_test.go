package vault_test

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/metrics"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/internal/connector/internal/services/vault"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/api"
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/shared"
	. "github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"io/ioutil"
	"strings"
	"testing"
	"text/template"
)

func TestNewVaultService(t *testing.T) {
	RegisterTestingT(t)
	vc := vault.NewConfig()

	// Enable testing against aws if the access keys are configured..
	if content, err := ioutil.ReadFile(shared.BuildFullFilePath(vc.AccessKeyFile)); err == nil && len(content) > 0 {
		vc.Kind = "aws"
	}
	Expect(vc.ReadFiles()).To(BeNil())

	tests := []struct {
		numSecrets   int // allow testing using aws vault with existing secrets
		config       *vault.Config
		wantErrOnNew bool
		skip         bool
	}{
		{
			config: &vault.Config{Kind: "tmp"},
		},
		{
			numSecrets: 92, // NOTE: change this to number of secrets actually in test AWS account after first failure
			config: &vault.Config{
				Kind:            "aws",
				AccessKey:       vc.AccessKey,
				SecretAccessKey: vc.SecretAccessKey,
				Region:          vc.Region,
			},
			skip: vc.Kind != "aws",
		},
		{
			config:       &vault.Config{Kind: "wrong"},
			wantErrOnNew: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.config.Kind, func(t *testing.T) {
			RegisterTestingT(t)
			svc, err := vault.NewVaultService(tt.config)
			Expect(err != nil).Should(Equal(tt.wantErrOnNew), "NewVaultService() error = %v, wantErr %v", err, tt.wantErrOnNew)
			if err == nil {
				if tt.skip {
					t.SkipNow()
				}
				happyPath(svc, tt.numSecrets)
			}
		})
	}
}

func happyPath(vault vault.VaultService, numSecrets int) {

	counter := 0
	err := vault.ForEachSecret(func(name string, owningResource string) bool {
		counter += 1
		return true
	})
	Expect(err).Should(BeNil())
	Expect(counter).Should(Equal(numSecrets))

	keyName := api.NewID()
	err = vault.SetSecretString(keyName, "hello", "thistest")
	Expect(err).Should(BeNil())

	value, err := vault.GetSecretString(keyName)
	Expect(err).Should(BeNil())
	Expect(value).Should(Equal("hello"))

	err = vault.DeleteSecretString(keyName)
	Expect(err).Should(BeNil())

	_, err = vault.GetSecretString("missing")
	Expect(err).ShouldNot(BeNil())

	err = vault.DeleteSecretString("missing")
	Expect(err).ShouldNot(BeNil())

	var builder strings.Builder
	err = tmpl.Execute(&builder, struct {
		GetCount         int
		SetCount         int
		TotalGetCount    int
		TotalDeleteCount int
	}{numSecrets + 1, 1, numSecrets + 2, 2})
	Expect(err).Should(BeNil())
	err = testutil.GatherAndCompare(prometheus.DefaultGatherer, strings.NewReader(builder.String()), vaultMetrics...)
	Expect(err).Should(BeNil())
}

var vaultMetrics []string = getMetricNames()

func getMetricNames() []string {
	names := []string{metrics.VaultServiceTotalCount, metrics.VaultServiceSuccessCount,
		metrics.VaultServiceErrorsCount, metrics.VaultServiceFailureCount}
	var result []string
	for _, m := range names {
		result = append(result, metrics.CosFleetManager+"_"+m)
	}
	return result
}

const expectedMetrics = `# HELP cos_fleet_manager_vault_service_errors_count count of user level errors (e.g. missing secrets) in the vault service
# TYPE cos_fleet_manager_vault_service_errors_count counter
cos_fleet_manager_vault_service_errors_count{operation="delete"} 1
cos_fleet_manager_vault_service_errors_count{operation="get"} 1
# HELP cos_fleet_manager_vault_service_success_count count of successful operations of vault service
# TYPE cos_fleet_manager_vault_service_success_count counter
cos_fleet_manager_vault_service_success_count{operation="delete"} {{.SetCount}}
cos_fleet_manager_vault_service_success_count{operation="get"} {{.GetCount}}
cos_fleet_manager_vault_service_success_count{operation="set"} {{.SetCount}}
# HELP cos_fleet_manager_vault_service_total_count total count of operations since start of vault service
# TYPE cos_fleet_manager_vault_service_total_count counter
cos_fleet_manager_vault_service_total_count{operation="delete"} {{.TotalDeleteCount}}
cos_fleet_manager_vault_service_total_count{operation="get"} {{.TotalGetCount}}
cos_fleet_manager_vault_service_total_count{operation="set"} {{.SetCount}}
`

var tmpl *template.Template = getMetricsTemplate()

func getMetricsTemplate() *template.Template {
	t, err := template.New("expected").Parse(expectedMetrics)
	if err != nil {
		panic(err)
	}
	return t
}
