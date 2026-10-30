# Configuration

You configure the authority with environment variables:

```sh
TIDEGATE_AUTHORITY__SERVER__LISTEN_ADDRESS=:8080
TIDEGATE_AUTHORITY__LOG__LEVEL=debug
```

These two make the authority listen on port 8080 on every network interface and log at the debug level.

Every variable starts with `TIDEGATE_AUTHORITY__`. Two underscores separate the parts of a name, like `SERVER` and `LISTEN_ADDRESS`. After the prefix, upper and lower case don't matter, in names or in values.

The authority checks every variable when it starts. If a value is invalid, or a variable starts with `TIDEGATE_AUTHORITY__` but isn't a setting, the authority stops and lists each problem:

```
Invalid config:
TIDEGATE_AUTHORITY__LOG__LEVL: unknown setting
TIDEGATE_AUTHORITY__LOG__LEVEL: unknown log level "loud", want debug, info, warn or error
```

## Settings

A setting keeps its default when its variable isn't set or is empty. `TIDEGATE_AUTHORITY__DATABASE__URL` is the only setting without a default. The pages on the [database](database.md), [TLS](tls.md) and [metrics](metrics.md) explain their settings.

| Variable | Default | Values |
|---|---|---|
| `TIDEGATE_AUTHORITY__SERVER__LISTEN_ADDRESS` | `127.0.0.1:8080` | A host and port, like `:8080` or `[::1]:8080` |
| `TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE` | None | Path to the certificate in PEM format, followed by any intermediate certificates |
| `TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE` | None | Path to the certificate's private key in PEM format |
| `TIDEGATE_AUTHORITY__DATABASE__URL` | None, required | A Postgres connection URL, like `postgres://tidegate:tidegate@host:5432/tidegate?sslmode=verify-full` |
| `TIDEGATE_AUTHORITY__DATABASE__MIGRATE_ON_START` | `false` | `true` or `false` |
| `TIDEGATE_AUTHORITY__METRICS__LISTEN_ADDRESS` | `127.0.0.1:9464` | A host and port for `/metrics`, like `:9464` |
| `TIDEGATE_AUTHORITY__LOG__LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `TIDEGATE_AUTHORITY__LOG__FORMAT` | `json` | `json` for log collectors, or `text`, which is easier to read locally |
