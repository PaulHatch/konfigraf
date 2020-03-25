<img src="https://www.konfigraf.com/logo.png" title="" alt="Konfigraf" width="200" height="174">

_Configuration as Code for Microservice Applications_

Konfigraf is a Postgres extension that allows you to store and manipulate data
in Git repositories stored in tables within the database. This is designed to
be used for storage of configuration data for applications and services.

For example, a file in the database can be accessed using the `get_file` function.

```sql

SELECT get_file('my-repository','app/config.json') AS config

```
