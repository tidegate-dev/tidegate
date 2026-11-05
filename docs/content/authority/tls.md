# TLS

The API serves HTTPS with a certificate and its key from these files:

```
/etc/tidegate/authority/tls/tls.crt
/etc/tidegate/authority/tls/tls.key
```

The names are the ones in a Kubernetes TLS Secret, so in Kubernetes you can mount the Secret at `/etc/tidegate/authority/tls`. Only the user the authority runs as should be able to read the key file. If your files are somewhere else, set their paths:

```sh
TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE=/srv/tls/cert.pem \
TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE=/srv/tls/key.pem \
authority
```

If a file is missing, the authority doesn't start and names the file. Only `authority` without a command, which serves the API, reads the files. `authority migrate` and `authority create-initial-admin` work without them.

To serve plain HTTP, for example behind a proxy that terminates TLS, set `TIDEGATE_AUTHORITY__SERVER__TLS__DISABLED=true`. The authority then logs a warning when it starts. If you also set the path of the certificate or the key, it doesn't start.

Note that the connection from the proxy to the authority is then unencrypted. Set `TIDEGATE_AUTHORITY__SERVER__LISTEN_ADDRESS` so that only the proxy reaches the API, for example `127.0.0.1:8080` when the proxy runs on the same host or in the same Pod. The container image listens on every interface by default.

The certificate's key needs at least 2048 bits for RSA or 224 bits for an elliptic curve, the minimum NIST sets for use through 2030. The authority doesn't start with a shorter key.

When you replace the files, the authority uses the new certificate for new connections within a minute, without a restart. If the new files don't hold a certificate and its key, or the key is too short, it keeps the old one and logs an error. The [metrics](metrics.md) say when the certificate in use expires. They always use plain HTTP.
