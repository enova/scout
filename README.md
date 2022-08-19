# Scout
![ci-status](https://github.com/enova/scout/workflows/CI/badge.svg)

Scout is a daemon for listening to a set of SNS topics and enqueuing anything it
finds into sidekiq jobs. It's meant to extract processing of SQS from the rails
apps that increasingly need to do so.

## Usage

```
NAME:
   scout - SQS Listener
Poll SQS queues specified in a config and enqueue Sidekiq jobs with the queue items.
It gracefully stops when sent SIGTERM.

USAGE:
   scout [global options] command [command options] [arguments...]

VERSION:
   v1.6.0

COMMANDS:
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config FILE, -c FILE       Load config from FILE, required
   --freq N, -f N               Poll SQS every N milliseconds (default: 100)
   --log-level value, -l value  Sets log level. Accepts one of: debug, info, warn, error
   --json, -j                   Log in json format
   --help, -h                   show help
   --version, -v                print the version
```

## Configuration

The configuration requires 3 distinct sets of information. It needs information
about how to connect to redis to enqueue jobs, credentials to talk to AWS and
read SQS, and a mapping from SNS topics to sidekiq worker classes in the
application. The structure looks like this.

```yaml
redis:
  host: "localhost:9000"
  namespace: "test"
  queue: "background"
  #
  password: "someoptionalpassword"
aws:
  access_key: "super"
  secret_key: "secret"
  region: "us-best"
queue:
  name: "myapp_queue"
  topics:
    foo-topic: "FooWorker"
    bar-topic: "BazWorker"
```

None of this information is actually an example of anything other than the
strucure of the file, so if you copy paste it you'll probably be disappointed.

## Versioning

Scout uses tagged commits that are compatible with go modules. The first module
aware version of scout is version `v1.5.0`. We recommend that you also use go
modules to guard against unexpected updates.

For legacy systems not using go modules, you can import using gopkg.in to pin
to version 1. The import path is `gopkg.in/enova/scout.v1`.

## Development

Scout uses go modules to manage it's dependencies, so you should clone it to a
location outside your `GOPATH`. At that point all the standard toolchain commands
do what they say on the box.

### Testing

The normal test suite can be run as expected with go test. There are also two
tagged files with expensive integration tests that require external services.
They can be run as follows

```
 [FG-386] scout > go test -run=TestSQS -v -tags=sqsint
=== RUN   TestSQS_Init
--- PASS: TestSQS_Init (3.84s)
=== RUN   TestSQS_FetchDelete
--- PASS: TestSQS_FetchDelete (3.58s)
    PASS
ok     github.com/enova/scout  7.422s
 [FG-386] scout > go test -run=TestWorker -v -tags=redisint
=== RUN   TestWorker_Init
--- PASS: TestWorker_Init (0.00s)
=== RUN   TestWorker_Push
--- PASS: TestWorker_Push (0.00s)
PASS
ok      github.com/enova/scout  0.013s
```

The tests themselves (found in `sqs_client_test.go` and `worker_client_test.go`)
explain what is required to run them. In particular, the SQS integration tests
require that you provide AWS credentials to run them.
