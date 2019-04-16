# Run plugin

You will need to add a configuration file named `plugin-run.yaml` with the following structure:

```yaml
Commands:
- Name: "test"
  Decription: "Run a gogo test, takes args like ruglar gogo test command"
  Command: "gogo test"
  AlowedUsers:
  - antoine@aporeto.com
- Name: ls
  Decription: "It list files, you can pass args"
- Name: sh
  AllowedUsers:
  - cyril@aporeto.com
```

`name`: The name or the command to run.
`description`: A description of the command.
`command` is the command to run in lieu of `name` if provided. So you can make aliases.
`AllowsedUsers` if present is the list of allowed users.
