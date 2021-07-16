# Access Control 
## Allow List Configurations

### Adding organisations and users to the allow list

To configure this list, you'll need to have the user's username and/or their organisation id.

The username is the account in question.

To get the org id:
- Login to `cloud.redhat.com/openshift/token` with the account in question.
- Use the supplied command to login to `ocm`,
- Then run `ocm whoami` and get the organisations id from `external_id` field.

### Max allowed instances
If the instance limit control is enabled, the service will enforce the `max_allowed_instances` configuration as the limit to how many instances (i.e. Kafka) a user can create. This configuration can be specified per user or per organisation in the allow list configuration. If not defined there, the service will take the default `max_allowed_instances` into account instead.

Precedence of `max_allowed_instances` configuration: Org > User > Default.

>NOTE: Instance limit control is disabled in the development environment.
### Deny List Configurations

Users can be denied access to the service explicitly by adding their usernames in [the list of denied users](config/deny-list-configuration.yaml).

The username is the account in question.

>NOTE: Once a user is in the deny list, all Kafkas created by this user will be deprovisioned.
