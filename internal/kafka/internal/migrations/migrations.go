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
	// We need to disable SQL Prepared Statements when performing Kafka DB
	// migrations due to caching issues with them in gorm that end up causing
	// errors. It seems the reasons for the issues might be a potential bug in
	// gorm, as when using previous gorm versions (v1.21.7) the issues didn't
	// happen.
	// A GitHub issue has been opened to the gorm project
	// reporting this: https://github.com/go-gorm/gorm/issues/5737
	// If it is confirmed to be an issue and gorm project fixes it we should
	// update this code accordingly when the fix is applied.
	dbConfigWithDisabledPreparedStatements := dbConfig.DeepCopy()
	dbConfigWithDisabledPreparedStatements.EnablePreparedStatements = false
	return db.NewMigration(dbConfigWithDisabledPreparedStatements, &gormigrate.Options{
		TableName:      "migrations",
		IDColumnName:   "id",
		IDColumnSize:   255,
		UseTransaction: false,
	}, migrations)
}
