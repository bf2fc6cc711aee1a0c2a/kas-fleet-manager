#!/bin/bash

# This bash script is meant to be run only in a development environment.
# DO NOT run this in production for any reason. There are risks assoicated with running SQL commands
# based off of defined variables in bash.
# Additionally, because the password for the user is passed through the command line, its possible
# for other users on the system to see the password in `ps` output.

set -e

USER=$(cat secrets/db.user)
PASSWORD=$(cat secrets/db.password)
DBNAME=$(cat secrets/db.name)

# Check if the user exists, this psql command will return 1 if it does, 0 if not
DB_USER_EXISTS=$(psql -tAc "SELECT 1 FROM pg_roles WHERE rolname='$USER';")
if [[ $DB_USER_EXISTS -ne 1 ]]; then
  # Create the user
  psql -c "CREATE USER \"$USER\" WITH PASSWORD '$PASSWORD';"
fi

# Check if the database exists, this psql command with exit 1 if not
if ! psql -lqt | cut -d \| -f 1 | grep -qw uhc; then
  createdb -O "$USER" "$DBNAME"
fi
