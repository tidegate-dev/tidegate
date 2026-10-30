# TLS

The API serves HTTPS when you set a certificate file and a key file:

```sh
TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE=/etc/tidegate/tls.crt \
TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE=/etc/tidegate/tls.key \
authority
```

Without them, it serves plain HTTP. If you set only one of them, the authority doesn't start.

When you replace the files, the authority uses the new certificate for new connections within a minute, without a restart. If the new files don't hold a certificate and its key, it keeps the old one and logs an error. The [metrics](metrics.md) say when the certificate in use expires. They always use plain HTTP.
