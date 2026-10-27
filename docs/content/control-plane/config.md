# Configuration

You configure the control plane with environment variables:

```sh
TIDEGATE_CP__SERVER__LISTEN_ADDRESS=:8080
TIDEGATE_CP__LOG__LEVEL=debug
```

These two make the control plane listen on port 8080 on every network interface and log at the debug level.

Every variable starts with `TIDEGATE_CP__`, where CP stands for control plane. Two underscores separate the parts of a name, like `SERVER` and `LISTEN_ADDRESS`. After the prefix, upper and lower case don't matter, in names or in values.

The control plane checks every variable when it starts. If a value is invalid, or a variable starts with `TIDEGATE_CP__` but isn't a setting, the control plane stops and lists each problem:

```
Invalid config:
TIDEGATE_CP__LOG__LEVL: unknown setting
TIDEGATE_CP__LOG__LEVEL: unknown log level "loud", want debug, info, warn or error
```

## Settings

A setting keeps its default when its variable isn't set or is empty.

The control plane serves its metrics at `/metrics` on their own address, in the Prometheus text format. That way you can make the API public and keep the metrics internal.

| Variable | Default | Values |
|---|---|---|
| `TIDEGATE_CP__SERVER__LISTEN_ADDRESS` | `127.0.0.1:8080` | A host and port, like `:8080` or `[::1]:8080` |
| `TIDEGATE_CP__METRICS__LISTEN_ADDRESS` | `127.0.0.1:9464` | A host and port for `/metrics`, like `:9464` |
| `TIDEGATE_CP__LOG__LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `TIDEGATE_CP__LOG__FORMAT` | `json` | `json` for log collectors, or `text`, which is easier to read locally |
