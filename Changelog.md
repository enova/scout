# Scout Changes

1.3
----------
- Rewrite the SQS integration to the use the AWS SDK instead of goamz

1.2
----------
- Save jobs in Redis with the Sidekiq `retry` flag set to `true`

1.1
----------

- Remove the `--quiet` flag in favor of `--log-level` which defaults to `INFO`
- Move some of the more verbose logging to `DEBUG` level logs
- Log full message body after parsing it

