# Facts plugin

You can teach some facts and ask them later


learn "something"
remind "something" to "someone" | remind "me" "something"
list-facts
add-pattern

ex:

m: @bot learn "How to reset your password"
b: @m glad to learn something ! Can you tell me "How to reset my password"?
m: @bot Go to the webiste and click the button.
b: @m: Ok got it, and which patterns should I use to trigger this fact?
m: @bot How to reset my password || How to reset my ldap password || How to reset my nuage password.
b: @bot Ok I will explain people "How to reset your password" for such patterns.


m: @bot can you tell @zob How to reset his password?
b: Sure, @zob here is how to reset your passord: "Go to the webiste and click the button."

Step 1:

    Follow a conversation

Step 2:
    Create a db scheme

type Fact struct {
    patterns []string
    name string
    fact string
}

ex:

name "How to reset your password"
patters "How to reset my password"
fact: "do this"


Latter

use kkn / stemmizer to add more patterns and do some knn clustering.
