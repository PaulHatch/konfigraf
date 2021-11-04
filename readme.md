<img src="https://www.konfigraf.com/logo.png" title="" alt="Konfigraf" width="200" height="174">

[![PGXN version](https://badge.fury.io/pg/konfigraf.svg)](https://badge.fury.io/pg/konfigraf)
[![Build](https://github.com/paulhatch/konfigraf/actions/workflows/build.yaml/badge.svg)](https://github.com/paulhatch/konfigraf/actions)

_Git Based Application Configuration_

Konfigraf is a Postgres extension that allows you to store and manipulate data
in Git repositories stored in tables within the database. This is designed to
be used for storage of configuration data for applications and services.

Example usage:

```sql

-- A repository must be created before any other actions are possible
SELECT create_repository('my-repository')

-- Files can be created or updated using the commit_file function
SELECT commit_file('my-repository', 'app/config.json','{"value": [1,2,3]}','John Doe','Set Value', 'john.d@example.com');

-- Return the file as a string
SELECT get_file('my-repository','app/config.json') AS config

-- List files for a given path as an array of strings
SELECT list_files('my-repository','app');

-- Since these are just SQL functions, they can be combined with queries

-- Here we update one value in a configuration
SELECT commit_file('my-repository', 'app/config.json',get_file('my-repository','app/config.json')::jsonb || '{"max":42}'::jsonb,'John Doe','Update max value', 'john.d@example.com');

-- Return a single value from a configuration
SELECT get_file('my-repository','app/config.json')::jsonb->'max' AS maximum

```

## Merge


## Performance

Konfigraf is designed primarily for use in the context of a user updating
application configuration, generally a low-volume operation.