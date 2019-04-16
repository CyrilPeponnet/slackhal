## Github plugin

You will need to add a configuration file named `plugin-github.yaml` with the following structure:

```yaml
Repositories:
  - Name: repo/.*
    Branches:
      - master
    Channels:
      - general
```

The `Name` must be a valid regular expression based on the repository name (ex: `myuser/project` or `myuser/.*`).

The `Branches` contains the list of branches to follow, you must set at least one.

The `Channels` contains the list of slack channel to send the message to.

The plugin handler the auto reload of the configuration file upon changes.

For all of this to work you will need to define a web hook service in your Repository in Github like:

```console
http://mybot.domain.local:8080/github
```

The `8080` port can be changed in the `slackhal` core.
