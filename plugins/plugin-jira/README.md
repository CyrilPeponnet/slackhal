# Jira plugin

Will passively react to ``#XXX-123` issue pattern in chats and show a summary of the issues.

Also provide a handler for Jira web hook `/jira` to notify issue creation according to what is configured in the `yaml` file.

## Configuration file

A configuration file is required for this plugin to work:

It must be named `plugin-jira.yaml` and contains:

```yaml
Server:
  url: <jira base url>
  username: <username>
  password: <password>
Notify:
 - Name: <name>
   Channels:
      - chan1
      - chan2
```

The plugin handler the auto reload of the configuration file upon changes.
