# Run plugin

You will need to add a configuration file named `plugin-run.yaml` with the following structure:

```yaml
Commands:
- Name: "test"
  Decription: "Run a du"
  Command:
  - du
  - -ha
  AlowedUsers:
  - alice@acme.tld
- Name: ls
  Decription: "It list files, you can pass args"
- Name: sh
  AllowedUsers:
  - bob@acme.tld
```

`name`: The name or the command to run.
`description`: A description of the command.
`command` is the command to run in lieu of `name` if provided. So you can make aliases.
`AllowsedUsers` if present is the list of allowed users.

During execution you can use the following env var:

- USER
