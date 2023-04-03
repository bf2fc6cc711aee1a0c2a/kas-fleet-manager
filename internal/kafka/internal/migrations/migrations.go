package migrations

import (
	"github.com/bf2fc6cc711aee1a0c2a/kas-fleet-manager/pkg/db"
	"github.com/go-gormigrate/gormigrate/v2"
)

// gormigrate is a wrapper for gorm's migration functions that adds schema versioning and rollback capabilities.
// For help writing migration steps, see the gorm documentation on migrations: https://gorm.io/docs/migration.html

// Migration rules:
//
//  1. IDs are numerical timestamps that must sort ascending.
//     Use YYYYMMDDHHMM w/ 24 hour time for format
//     Example: August 21 2018 at 2:54pm would be 201808211454.
//
//  2. Include models inline with migrations to see the evolution of the object over time.
//     Using our internal type models directly in the first migration would fail in future clean installs.
//
//  3. Migrations must be backwards compatible. There are no new required fields allowed.
//     See $project_home/db/README.md
//
// 4. Create one function in a separate file that returns your Migration. Add that single function call to this list.
var migrations = []*gormigrate.Migration{
	addKafkaRequest(),
	addClusters(),
	updateKafkaMultiAZTypeToBoolean(),
	addKafkabootstrapServerHostType(),
	addClusterStatus(),
	addKafkaOrganisationId(),
	addLeaderLease(),
	addFailedReason(),
	addConnectors(),
	addKafkaPlacementId(),
	addConnectorClusters(),
	addKafkaSubscriptionId(),
	addClusterIdentityProviderID(),
	addKafkaSsoClientIdAndSecret(),
	addKafkaOwnerAccountId(),
	addKafkaVersion(),
	connectorApiChanges(),
	addMissingIndexes(),
	addClusterStatusIndex(),
	addKafkaWorkersInLeaderLeases(),
	renameDeletingKafkaLeaseType(),
	addClusterDNS(),
	connectorMigrations20210518(),
	addExternalIDsToSpecificClusters(),
	addKafkaConnectionSettingsToConnectors(),
	addClusterProviderInfo(),
	changeKafkaDeleteStatusToDeleting(),
	addKafkaQuotaTypeColumn(),
	addConnectorTypeChannel(),
	addRoutes(),
	addKafkaDNSWorkerLease(),
	addClusterAvailableStrimziVersions(),
	addKafkaUpgradeFunctionalityRelatedFields(),
	renameKafkaVersionField(),
	updateDesiredStrimziVersions(),
	addKafkaInstanceTypeColumn(),
	addKafkaCanaryServiceAccountColumns(),
	addKafkaNamespaceColumn(),
	migrateOldKafkaNamespace(),
	migrateOldKafkaNamespaceCreatedDuringDeployment(),
	replaceAllowListWithQuotaManagementList(),
	resetOldIngressControllerRoutes(),
	resetCanaryServiceAccountWithTwoDashes(),
	resetCanaryServiceAccountForTwoInstances(),
	addKafkaFailedWorkerLease(),
	resetCanaryServiceAccountForAffectedInstances(),
	addClusterSupportedInstanceType(),
	addKafkaReauthenticationEnabledColumn(),
	addKafkaIBPVersionRelatedFields(),
	addKafkaRoutesCreationIdColumn(),
	addKafkaStorageSize(),
	addClusterServiceAccountId(),
	addClusterServiceClientSecret(),
	addKafkaSizeId(),
	dropKafkaSsoClientIdAndSecret(),
	addAdminApiServerURL(),
	addKafkaCloudAccountIdMarketplaceFields(),
	addKafkaBillingModel(),
	addClusterDynamicCapacityInfo(),
	addDynamicScaleUpWorkerToLeaderLeases(),
	addCleanupClusterExternalResourcesWorkerToLeaderLeases(),
	addDeprovisioningClusterWorkerToLeaderLeases(),
	addDynamicScaleDownWorkerToLeaderLeases(),
	removeTheWronglyAutoCreatedClusterInStageEnvironmentWithID_cdhunvd8igjhbi0nmtt0(),
	addDesiredKafkaBillingModel(),
	renameKafkaBillingModelColumn(),
	addExpiresAtToKafkaRequest(),
	addClusterOrgIdClusterTypeColumns(),
	addDefaultValueForClusterTypeColumn(),
	addKafkaPromotionFields(),
	removeWronglyCreatedEnterpriseClusterInProd(),
	addAccessKafkasViaPrivateNetworkColumnInClustersTable(),
	addKafkaPromoteWorkerInLeaderLeases(),
	updateExpiresAtZeroValueFromKafkaRequests(),
	renameKafkaStorageSizeColumn(),
	addKafkaDomainCertificateManagementInfoInKafkaRequestsTable(),
	addKafkasRoutesTLSCertificateManagerInLeaderLeases(),
	addDistributedLockTable(),
}

func New(dbConfig *db.DatabaseConfig) (*db.Migration, func(), error) {
	return db.NewMigration(dbConfig, &gormigrate.Options{
		TableName:      "migrations",
		IDColumnName:   "id",
		IDColumnSize:   255,
		UseTransaction: false,
	}, migrations)
}
