# Plugin CI

This is plugin is designed to post a CI status on a desired channel and report any failure cases.

It will expose a simple REST API.


X pushed Y commits to Z:branch
Build: (running|failed|pass)
Tests: (running|failed|pass)

type CIState struct {
    Repository string
    Branch string
    Commit string
    Status {
        Build {
            Link string
            State string
        }
        Test {
            Link string
            State string
        }
    }
}


Key: Repository + Commit Hash
Value: TimeStamp of posted message

We need to find a way to store the TimeStamp of the first message sent.
