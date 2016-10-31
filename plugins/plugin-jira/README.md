# Jira plugin

Will passively react to ``#XXX-123` issue pattern in chats and show a summary of the isse.

Also provide a handler for jira webhook `/jira` to notify issue creation according to what is configured in the yaml file.


## Configuration file
A configuration file is required for this plugin to work:

It must be named `plugin-jira.yaml` and contains:

``
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
