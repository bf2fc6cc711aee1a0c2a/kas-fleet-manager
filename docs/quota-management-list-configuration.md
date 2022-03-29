# Quota Control
## Quota Management List Configurations

The type and the quantity of kafka instances a user can create is controlled via the 
[Quota Management List](../config/quota-management-list-configuration.yaml).
If a user is not in the _Quota Management List_, only DEVELOPER kafka instances will be allowed.

The difference between STANDARD and DEVELOPER instance is its lifespan: DEVELOPER instance will be deleted automatically after 
48 hours.

### Adding organisations and users to the Quota Management List

To configure this list, you'll need to have the user's username and/or their organisation id.

The username is the account in question.

To get the org id:
- Login to `cloud.redhat.com/openshift/token` with the account in question.
- Use the supplied command to login to `ocm`,
- Then run `ocm whoami` and get the organisations id from `external_id` field.

### Max allowed instances
If the instance limit control is enabled, the service will enforce the `max_allowed_instances` configuration as the 
limit to how many instances (i.e. Kafka) a user can create. This configuration can be specified per user or per 
organisation in the quota list configuration. If not defined there, the service will take the default 
`max_allowed_instances` into account instead.

Precedence of `max_allowed_instances` configuration: Org > User > Default.
