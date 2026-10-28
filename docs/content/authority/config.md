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

A setting keeps its default when its variable isn't set or is empty.

The authority serves its metrics at `/metrics` on their own address, in the Prometheus text format. That way you can make the API public and keep the metrics internal.

The authority keeps its data in Postgres, and `TIDEGATE_AUTHORITY__DATABASE__URL` is the only setting without a default. Before you start a new version of the authority, run `authority migrate` with the same variables. It brings the database schema up to date and exits. The authority refuses to start if the schema is older than it expects. With `TIDEGATE_AUTHORITY__DATABASE__MIGRATE_ON_START` set to `true`, the authority migrates when it starts instead. Its database user then needs permission to change the schema.

Set `sslmode=verify-full` in the database URL, so the authority only connects to a Postgres server with a valid certificate. Without an `sslmode`, the connection may fall back to plain text, and the authority logs a warning when it starts.

The API serves HTTPS when you set both a certificate file and a key file. Without them, it serves plain HTTP. If you set only one of them, the authority doesn't start. The authority reads both files when it starts, so restart it after you replace them. The metrics always use plain HTTP.

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

## Running under systemd

If you run the authority as a systemd service, give its unit `Type=notify`:

```ini
[Service]
Type=notify
ExecStart=/usr/local/bin/authority
```

systemd then counts the authority as started once it accepts connections, not as soon as the process runs. `systemctl start` waits until then, and units ordered after the authority don't start too early. Outside systemd, the authority starts and stops as usual.
