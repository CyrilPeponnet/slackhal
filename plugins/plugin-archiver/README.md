# Logger plugin

Will log followed channel to botbot-me postgresql database.

## Configuration file
A configuration file is required for this plugin to work:

It must be named `plugin-logger.yaml` and contains:

``
Database:
  url: <db base url>
  username: <username>
  password: <password>
Server:
  base_url: <base usr of the botbot-me instance>
```
