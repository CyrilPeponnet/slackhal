## Github plugin

You will need to add a configuraiton file named `plugin-github.yaml` with the following structure

```
Repositories:
  - Name: repo/.*
    Branches:
      - master
    Channels:
      - general
```

The `Name` must be a valid regexp based on the repoistory name (ex: `myuser/project` or `myuser/.*`).

The `Branches` contains the list of branches to follow, you must set at least one.

The `Channels` contains the list of slack channel to send the message to.


For all of this to work you will need to define a webhook service in your Repository in github like:

```
http://mybot.domain.local:8080/github
```

The `8080` port can be changed in the `slackhal` core.
