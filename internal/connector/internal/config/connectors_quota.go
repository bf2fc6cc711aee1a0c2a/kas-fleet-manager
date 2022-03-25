package config

// NamespaceQuota has resource limits for namespaces
type NamespaceQuota struct {
	Connectors     int32  `yaml:"connectors,omitempty"`
	MemoryRequests string `yaml:"memory-requests,omitempty"`
	MemoryLimits   string `yaml:"memory-limits,omitempty"`
	CPURequests    string `yaml:"cpu-requests,omitempty"`
	CPULimits      string `yaml:"cpu-limits,omitempty"`
}

// Quotas has limits for various resource types, e.g. namespaces
// other resource limits can be added in the future,
// e.g clusters with limits on namespaces, connectors with limits on catalogs, etc.
type Quotas struct {
	NamespaceQuota NamespaceQuota `yaml:"namespace-quota,omitempty"`
}

// ConnectorsQuotaProfile is a named quotas configuration
type ConnectorsQuotaProfile struct {
	Name   string `yaml:"profile-name"`
	Quotas Quotas `yaml:"quotas"`
}

type ConnectorsQuotaProfileList []ConnectorsQuotaProfile

type ConnectorsQuotaProfileMap map[string]Quotas
