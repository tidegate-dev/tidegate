# TLS

The API serves HTTPS with a certificate and its key from these files:

```
/etc/tidegate/authority/tls/tls.crt
/etc/tidegate/authority/tls/tls.key
```

The names are the ones in a Kubernetes TLS Secret, so in Kubernetes you can mount the Secret at `/etc/tidegate/authority/tls`. If your files are somewhere else, set their paths:

```sh
TIDEGATE_AUTHORITY__SERVER__TLS__CERT_FILE=/srv/tls/cert.pem \
TIDEGATE_AUTHORITY__SERVER__TLS__KEY_FILE=/srv/tls/key.pem \
authority
```

If a file is missing, the authority doesn't start and names the file. Only `authority` without a command, which serves the API, reads the files. `authority migrate` and `authority create-initial-admin` work without them.

To serve plain HTTP, for example behind a proxy that terminates TLS, set `TIDEGATE_AUTHORITY__SERVER__TLS__DISABLED=true`. The authority then logs a warning when it starts. If you also set the path of the certificate or the key, it doesn't start.

Note that the connection from the proxy to the authority is then unencrypted, including the passwords and tokens of logins. Set `TIDEGATE_AUTHORITY__SERVER__LISTEN_ADDRESS` so that only the proxy reaches the API, for example `127.0.0.1:8080` when the proxy runs on the same host or in the same Pod. The container image listens on every interface by default.

The certificate's key needs at least 2048 bits for RSA or 224 bits for an elliptic curve, the minimum NIST sets for use through 2030. The authority doesn't start with a shorter key.

When you replace the files, the authority uses the new certificate for new connections within a minute, without a restart. If the new files don't hold a certificate and its key, the key is too short or other users can access the key file, it keeps the old one and logs an error. The [metrics](metrics.md) say when the certificate in use expires. They always use plain HTTP.

## Who can read the key

The authority doesn't start if users other than root and its own user can read or change the key file. In Kubernetes, mount the Secret like this:

```yaml
spec:
  securityContext:
    fsGroup: 65532
  containers:
    - name: authority
      volumeMounts:
        - name: tls
          mountPath: /etc/tidegate/authority/tls
          readOnly: true
  volumes:
    - name: tls
      secret:
        secretName: authority-tls
        defaultMode: 0440
```

The Secret's files belong to root. `fsGroup` gives them the group 65532, the user ID of the container image, and `defaultMode` lets only the owner and the group read them. Without `defaultMode`, every user in the container can read the key.

Outside Kubernetes, give the key file to the user the authority runs as, with mode 0600 or 0400. If root owns the file, mode 0640 or 0440 works too, and the authority reads it through the file's group. Note that the authority doesn't check who else is in that group. The certificate file can have any mode.
