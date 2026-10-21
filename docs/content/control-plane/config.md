# Configuration

You configure the control plane with environment variables. Every variable starts with `TIDEGATE_CP__`, where CP stands for control plane. Two underscores also separate the levels of a setting.

```sh
TIDEGATE_CP__SERVER__LISTEN_ADDRESS=:8080
TIDEGATE_CP__LOG__LEVEL=debug
```

These two variables make the control plane listen on port 8080 on every interface and log at the debug level.

| Variable | Default | Values |
|---|---|---|
| `TIDEGATE_CP__SERVER__LISTEN_ADDRESS` | `127.0.0.1:8080` | A host and port, like `:8080` or `[::1]:8080` |
| `TIDEGATE_CP__LOG__LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `TIDEGATE_CP__LOG__FORMAT` | `json` | `json` for log collectors, or `text`, which is easier to read locally |

A setting keeps its default when its variable isn't set or is empty. After the prefix, upper and lower case don't matter, in names or in values.

The default listen address only accepts connections from the same machine. To reach the control plane from elsewhere, set it to `:8080`. The container image already does that.

The control plane checks every variable when it starts. If a value is invalid, or a variable that starts with `TIDEGATE_CP__` isn't a known setting, it stops and lists each problem with the variable's name:

```
Invalid config:
TIDEGATE_CP__LOG__LEVL: unknown setting
TIDEGATE_CP__LOG__LEVEL: unknown log level "loud", want debug, info, warn or error
```
