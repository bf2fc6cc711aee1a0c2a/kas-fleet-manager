#!/bin/bash -ex

export AWS_ACCESS_KEY_ID="$AWS_ACCESS_KEY"
export AWS_SECRET_ACCESS_KEY="$AWS_SECRET"

FILENAME="$(date +%Y-%m-%d_%H-%M-%S)"
KUBECONFIG_LOCATION="/usr/local/bin/.kubeconfig"

DUMP_FILENAME="/usr/local/bin/$FILENAME"

oc login --server="$SERVER_URL" -u "$ADMIN_LOGIN" -p "$ADMIN_PASSWORD" --kubeconfig "$KUBECONFIG_LOCATION"

oc --kubeconfig "$KUBECONFIG_LOCATION" project "$BACKUP_PROJECT"

oc --kubeconfig "$KUBECONFIG_LOCATION" exec "$DB_POD_NAME" -- bash -c "pg_dump $DB_NAME -U $DB_USER > /var/lib/pgsql/data/$FILENAME"

oc --kubeconfig "$KUBECONFIG_LOCATION" cp "$DB_POD_NAME:/var/lib/pgsql/data/$FILENAME" "$DUMP_FILENAME"

aws s3 cp "$DUMP_FILENAME" "s3://$S3_BUCKET_NAME"

oc --kubeconfig "$KUBECONFIG_LOCATION" exec "$DB_POD_NAME" -- bash -c "rm /var/lib/pgsql/data/$FILENAME"
