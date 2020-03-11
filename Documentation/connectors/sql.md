# Authentication through SQL

## Overview

Connector allows dex to look up users and groups in a SQL database through user-provided queries.

## Configuration

The following is an example of a configuration for `examples/config-dev.yaml`:

```yaml
connectors:
- type: sql
  # Required field for connector id.
  id: internal-authentication-database
  # Required field for connector name.
  name: Internal Auth Database
  config:
	database: postgres
	connection: host=authdb-slave database=auth password=secret username=auth-readonly
	prompt: "ACME Corp login"

	# Login - gets 'username' and 'password' as positional parameters and should return true if the user exists (false or no rows will reject)
	login: |
	  SELECT COUNT(*) = 1
	  FROM auth
	  WHERE username = :1 AND password = crypt( :2, password)

	# Get user information - gets 'username' and must return 'UserId', 'Username', 'PreferredUsername', 'Email' and 'EmailVerified' (bool)
	# This is also used for refresh, so it has to return zero rows if the user ceases to exist
	get-identity: |
	  SELECT username, username, username, username + "@example.com", true
	  FROM auth
	  WHERE username = :1

	# Optional - setting to empty string will give empty list of groups
	groups: |
	  SELECT groupName FROM group_members WHERE username = :1
```

 * The groups-query should return one row for each group, with just one column
   as the group name.
 * Dex' config-loader mocks around with `$xxx`-things (replaces with values
   from env or blanks them), which means it swallows Postgres' positional
   parameters. As a quick hack, all ` @` and ` :` will be replaced with ` ?`
   when running with a postgres backend.

