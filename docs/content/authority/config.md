# Configuration

You configure the authority with environment variables:

```sh
TIDEGATE_AUTHORITY__EXTERNAL_API__LISTEN_ADDRESS=:8080 \
TIDEGATE_AUTHORITY__LOG__LEVEL=debug \
authority
```

This makes the external API, which the CLI and the browser call, listen on port 8080 on every network interface, and sets the log level to debug.

Every variable starts with `TIDEGATE_AUTHORITY__`. Two underscores separate the parts of a name, like `EXTERNAL_API` and `LISTEN_ADDRESS`. After the prefix, upper and lower case don't matter in names, or in values like `true` and `debug`. A path or the database URL is used as you write it.

The authority checks every variable when it starts. If a value is invalid, or a variable starts with `TIDEGATE_AUTHORITY__` but isn't a setting, the authority stops and lists each problem:

```
Invalid config:
TIDEGATE_AUTHORITY__LOG__LEVL: unknown setting
TIDEGATE_AUTHORITY__LOG__LEVEL: unknown log level "loud", want debug, info, warn or error
```

## Settings

A setting keeps its default when its variable isn't set or is empty. `TIDEGATE_AUTHORITY__DATABASE__URL` is the only setting without a default. The pages on the [database](database.md), [TLS](tls.md) and [metrics](metrics.md) explain their settings.

The authority serves two APIs, each on its own address and with its own certificate. Besides the external API, there is an internal API for tidegate's other components, like gateways. It has no services yet. Don't make its address reachable for users.

| Variable | Default | Values |
|---|---|---|
| `TIDEGATE_AUTHORITY__EXTERNAL_API__LISTEN_ADDRESS` | `127.0.0.1:8080` | A host and port, like `:8080` or `[::1]:8080` |
| `TIDEGATE_AUTHORITY__EXTERNAL_API__TLS__CERT_FILE` | `/etc/tidegate/authority/external-api-tls/tls.crt` | Path to the certificate in PEM format, followed by any intermediate certificates |
| `TIDEGATE_AUTHORITY__EXTERNAL_API__TLS__KEY_FILE` | `/etc/tidegate/authority/external-api-tls/tls.key` | Path to the certificate's private key in PEM format |
| `TIDEGATE_AUTHORITY__EXTERNAL_API__TLS__DISABLED` | `false` | `true` serves plain HTTP, for a proxy that terminates TLS |
| `TIDEGATE_AUTHORITY__INTERNAL_API__LISTEN_ADDRESS` | `127.0.0.1:8081` | A host and port, like `:8081` |
| `TIDEGATE_AUTHORITY__INTERNAL_API__TLS__CERT_FILE` | `/etc/tidegate/authority/internal-api-tls/tls.crt` | Path to the certificate in PEM format, followed by any intermediate certificates |
| `TIDEGATE_AUTHORITY__INTERNAL_API__TLS__KEY_FILE` | `/etc/tidegate/authority/internal-api-tls/tls.key` | Path to the certificate's private key in PEM format |
| `TIDEGATE_AUTHORITY__DATABASE__URL` | None, required | A Postgres connection URL, like `postgres://tidegate:tidegate@host:5432/tidegate?sslmode=verify-full` |
| `TIDEGATE_AUTHORITY__DATABASE__MIGRATE_ON_START` | `false` | `true` or `false` |
| `TIDEGATE_AUTHORITY__LOGIN__TOKEN_LIFETIME` | `12h` | A duration of at least `1s`, like `12h` or `30m` |
| `TIDEGATE_AUTHORITY__METRICS__LISTEN_ADDRESS` | `127.0.0.1:9464` | A host and port for `/metrics`, like `:9464` |
| `TIDEGATE_AUTHORITY__LOG__LEVEL` | `info` | `debug`, `info`, `warn` or `error` |
| `TIDEGATE_AUTHORITY__LOG__FORMAT` | `json` | `json` for log collectors, or `text`, which is easier to read locally |
