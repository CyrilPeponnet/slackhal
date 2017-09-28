# Archiver plugin

Will log followed channel to botbot-me postgresql database.

## Configuration file
A configuration file is required for this plugin to work:

It must be named `plugin-archiver.yaml` and contains:

```
Database:
  url: <db base url>
  username: <username>
  password: <password>
UI:
  url: <base usr of the botbot-me instance>
```

## Commands

Commands can be either called within the channel or using DM to the bot. In this case the channel name must be provided as an argument.

*NOTE:* For private channel you cannot use direct messages so the commands must be called from the channel directly.

- `log`: will start to log all conversation
- `no-log`: will stop to log
- `archive`: will display the link to archive and give the log state.
