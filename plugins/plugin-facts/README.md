# Facts plugin

You can teach some facts and ask them later

```console
m: @bot new-fact Age of the bot AS I never age. WHEN @bot how old are you OR How hold is the bot IN #random
```

Then

```console
m: @bot how old are you?
bot: I have no age, I'm forever young.
```

## Configuration file

A `yaml` named `plugin-facts.yaml` must be present with the following content:

```yaml
database:
  path: facts.db
```

Where `database.path` is the path of the fact database.
