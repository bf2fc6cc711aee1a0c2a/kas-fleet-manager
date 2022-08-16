package cloudproviders

import "strings"

// CloudProviderID represents a Cloud Provider ID
type CloudProviderID string

// CloudProviderID typed constants that can be used throughout the
// code to reference a specific Cloud Provider ID without duplicating
// its value in several places
const (
	AWS   CloudProviderID = "aws"
	GCP   CloudProviderID = "gcp"
	Azure CloudProviderID = "azure"

	Unknown CloudProviderID = ""
)

func (c CloudProviderID) String() string {
	return string(c)
}

var CloudPoviderIDToDisplayNameMapping map[CloudProviderID]string = map[CloudProviderID]string{
	AWS:   "Amazon Web Services",
	Azure: "Microsoft Azure",
	GCP:   "Google Cloud Platform",
}

type CloudProviderCollection struct {
	providers map[CloudProviderID]struct{}
}

// Contains returns whether a given `id` CloudProvider is contained
// in the collection
func (s *CloudProviderCollection) Contains(id CloudProviderID) bool {
	_, ok := s.providers[id]
	return ok
}

var defaultKnownCloudProviders = CloudProviderCollection{
	providers: map[CloudProviderID]struct{}{
		AWS:   {},
		Azure: {},
		GCP:   {},
	},
}

// KnownCloudProviders returns a CloudProviderCollection containing
// the Cloud Providers recognized by the Fleet Manager.
// Used to limit the providers that can be specified in the providers
// configuration file.
func KnownCloudProviders() CloudProviderCollection {
	return defaultKnownCloudProviders
}

// ParseCloudProviderID takes a Cloud Provider ID `id` string representation
// and returns its corresponding CloudProviderID typed representation.
// If it is not a recognized Cloud Provider ID by the Fleet Manager then
// the `Unknown` CloudProviderID is returned
func ParseCloudProviderID(id string) CloudProviderID {
	normalizedID := strings.ToLower(id)
	switch CloudProviderID(normalizedID) {
	case AWS:
		return AWS
	case GCP:
		return GCP
	case Azure:
		return Azure
	default:
		return Unknown
	}
}
