# Facts plugin

You can teach some facts and ask them later

```
m: @bot new-fact How old is the bot
bot: Ok @m let's do that! Can you define _How old is the bot_?
(type stop-learning to stop this learning session)
m: I have no age, I'm forever young.
bot: Got it @m. And now can you tell me list of pattern I should match for this fact (Use || as separator).
m: how old is the bot || what is the age of the bot || @bot how old are you
bot: All good! I'll keep that in mind.
bot: I now know 1 facts.
```

Then

```
m: @bot how old are you?
bot: I have no age, I'm forever young.
```

## Extra flags upon creation

`channel:lock` ->  restrict a fact to the current channel.

## Configuration file

A yaml named `plugin-facts.yaml` must be present with the following content:


```yaml
database:
  path: facts.db
```

Where `database.path` is the path of the fact database.
