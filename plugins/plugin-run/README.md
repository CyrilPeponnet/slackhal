# Run plugin

You will need to add a configuration file named `plugin-run.yaml` with the following structure:

```yaml
Commands:
- Name: "test"
  Decription: "Run a du"
  Command:
  - du
  - -ha
- Name: ls
  Decription: "It list files, you can pass args"
- Name: sh
```

`name`: The name or the command to run.
`description`: A description of the command.
`command` is the command to run in lieu of `name` if provided. So you can make aliases.

During execution you can use the following env var:

- USER
