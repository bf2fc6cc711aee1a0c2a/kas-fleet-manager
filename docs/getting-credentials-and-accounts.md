> NOTE: This document refers to Red Hat internal components

# Quota Management with Account Management Service (AMS) SKU

The [Account Management Service](https://api.openshift.com/?urls.primaryName=Accounts%20management%20service) manages users subscriptions. The fleet manager uses the service offered by AMS to manage quota. Quota comes in form of stock keeping unit (SKU) assigned to a given organisation. The process is the same as requesting SKU for addons which is described in the following link https://gitlab.cee.redhat.com/service/managed-tenants/-/blob/main/docs/tenants/requesting_skus.md.

# User Account & Organization Setup

1. Create new Red Hat organization and users belonging to this organization:
* Make sure that you are logged out from any existing `access.redhat.com` sessions. Use the `register new organization` flow to create an organization
* Create new users using user management UI. It is recommended to append or prefix usernames of your team members, e.g. `dffrench_control_plane` for `dffrench` username being a member of `Control Plane` team.
2. An AMS Role defined through the OCM Resources mechanism has been defined, available for developers of the KAS Fleet Manager to get the needed permissions
   to perform KAS Fleet Manager development. The definition of the role can be found in 
   https://gitlab.cee.redhat.com/service/uhc-account-manager/-/blob/master/pkg/api/roles/managed_kafka_service.go.
3. Users interested in developing KAS Fleet Manager should have this role assigned to their OCM account. This can be done using the OCM Resources
   mechanism. An example of assigning the AMS role previously mentioned to a user can be found [here](https://gitlab.cee.redhat.com/service/ocm-resources/-/blob/master/data/uhc-stage/users/msoriano_kafka_service.yaml).
4. KAS Fleet Manager developers also need AMS quota for installing specific OCM add-ons that KAS Fleet Manager requires. An example showing the required quotas for the addons can be found [here](https://gitlab.cee.redhat.com/service/ocm-resources/-/blob/master/data/uhc-stage/orgs/13640203.yaml).

# Vault
For automation (fleet-manager deployments and app-interface pipelines) it is required to use [vault](https://gitlab.cee.redhat.com/service/app-interface#manage-secrets-via-app-interface-openshiftnamespace-1yml-using-vault) for secrets management

# AWS
AWS accounts are required for provisioning development OSD clusters and to access AWS Route53 for Domain Name System.

Within the AWS account used to provision OSD clusters you must create
an AWS IAM user with the following requirements:
* This user needs at least Programmatic access enabled.
* This user must have the AdministratorAccess policy attached to it.
See https://docs.openshift.com/dedicated/osd_planning/aws-ccs.html#ccs-aws-customer-procedure_aws-ccs for more information.

NOTE: If you use OCM to provision them the name of the IAM user must be
named `osdCcsAdmin`.

# SSO authentication
Depending on whether interacting with public or private endpoints, the
authentication can be set up as follows:
* Red Hat SSO sso.redhat.com for public endpoints
* Red HAT SSO or MAS SSO for private endpoints. See this
[doc](https://gitlab.cee.redhat.com/service/app-interface/-/blob/master/docs/mas-sso/service-architecture/service-architecture.md)
to understand how kas-fleet-manager is using MAS-SSO for authentication

See [feature-flags](feature-flags.md#Keycloak) to understand flags used for authentication.

## sso.redhat.com service account
KAS Fleet Manager can also be ran with a dedicated sso.redhat.com service
account instead of using your own Red Hat account for the communication
between the KAS Fleet Manager and OCM. Instead of specifying an OCM offline
token, you will need to specify the service account's client id and secret
to KAS Fleet Manager.

To get a service account created in sso.redhat.com, open a help desk
requesting it to Red Hat IT.