# TLS

The API serves HTTPS when you set a certificate file and a key file:

```sh
TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE=/etc/tidegate/tls.crt
TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE=/etc/tidegate/tls.key
```

Without them, it serves plain HTTP. If you set only one of them, the authority doesn't start.

The authority reads both files when it starts, so restart it after you replace them.

The [metrics](metrics.md) always use plain HTTP.
