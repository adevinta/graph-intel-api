# Security Graph - Intel API

## Security Graph

Security Graph is a data architecture that provides real-time views of assets
and their relationships. Assets to be considered include software, cloud
resources and, in general, any piece of information that a security team needs
to protect.

Security Graph not only stores the catalog of active assets and the assets
assigned to teams. It also keeps a historical log of the multidimensional
relationships among these assets, including their attributes relevant to
security.

## Intel API

The "intel" API is a web service that exposes processed data from the Security
Graph. For instance, it exposes the Blast Radius score of a specific asset.

## Test

Execute the tests:

```
_script/test -cover ./...
```

`_script/test` makes sure the testing infrastructure is up and running and then
runs `go test` with the provided arguments. It also disables test caching and
avoids running multiple test programs in parallel.

Stop the testing infrastructure:

```
_script/clean
```

## Environment Variables

The following environment variables are **required**:

| Variable | Description | Example |
| --- | --- | --- |
| `GREMLIN_ENDPOINT` | Gremlin server endpoint | `ws://127.0.0.1:8182/gremlin` |

The following environment variables are **optional**:

| Variable | Description | Default |
| --- | --- | --- |
| `LOG_LEVEL` | Log level. Valid values: `info`, `debug`, `error`, `disabled` | `info` |
| `LISTEN_ADDR` | Listen address of graph-intel-api | `:8000` |
| `GREMLIN_AUTH_MODE` | Gremlin server authentication mode. Valid values: `plain`, `neptune_iam` | `plain` |
| `AWS_REGION` | AWS region | `eu-west-1` |
| `GREMLIN_RETRY_LIMIT` | Number of retries before a Gremlin query returns error | `5` |
| `GREMLIN_RETRY_DURATION` | Time to wait between Gremlin query retries | `5s` |
| `INTEL_RESOLVE_TIMEOUT_MS` | Query timeout in ms used when finding assets. If zero, no timeout is set | `60000` |
| `INTEL_BLAST_RADIUS_TIMEOUT_MS` | Query timeout in ms used when calculating the blast radius score. If zero, no timeout is set.| `60000` |

The directory `_env` in this repository contains some example configurations.

## Contributing

**This project is in an early stage, we are not accepting external
contributions yet.**

To contribute, please read the contribution guidelines in [CONTRIBUTING.md].


[CONTRIBUTING.md]: CONTRIBUTING.md
