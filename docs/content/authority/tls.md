# TLS

Each of the authority's [two APIs](config.md) serves HTTPS with its own certificate. The authority reads each certificate and its key from these files:

```
/etc/phemera/authority/external-api-tls/tls.crt
/etc/phemera/authority/external-api-tls/tls.key
/etc/phemera/authority/internal-api-tls/tls.crt
/etc/phemera/authority/internal-api-tls/tls.key
```

The names are the ones in a Kubernetes TLS Secret, so in Kubernetes you can mount one Secret at each folder. If your files are somewhere else, set their paths:

```sh
PHEMERA_AUTHORITY__EXTERNAL_API__TLS__CERT_FILE=/srv/tls/external.pem \
PHEMERA_AUTHORITY__EXTERNAL_API__TLS__KEY_FILE=/srv/tls/external-key.pem \
PHEMERA_AUTHORITY__INTERNAL_API__TLS__CERT_FILE=/srv/tls/internal.pem \
PHEMERA_AUTHORITY__INTERNAL_API__TLS__KEY_FILE=/srv/tls/internal-key.pem \
authority
```

If a file is missing, the authority doesn't start and names the file. Only `authority` without a command, which serves the APIs, reads the files. `authority migrate` and `authority create-initial-admin` work without them.

To serve the external API over plain HTTP, for example behind a proxy that terminates TLS, set `PHEMERA_AUTHORITY__EXTERNAL_API__TLS__DISABLED=true`. The authority then logs a warning when it starts. If you also set the path of the external API's certificate or key, it doesn't start. The internal API has no such setting and always serves HTTPS.

Note that the connection from the proxy to the authority is then unencrypted, including the passwords and tokens of logins. Set `PHEMERA_AUTHORITY__EXTERNAL_API__LISTEN_ADDRESS` so that only the proxy reaches the external API, for example `127.0.0.1:8080` when the proxy runs on the same host or in the same Pod. The container image listens for the external API on every interface by default.

A certificate's key needs at least 2048 bits for RSA or 224 bits for an elliptic curve, the minimum NIST sets for use through 2030. The authority doesn't start with a shorter key.

When you replace a certificate's files, the authority uses the new certificate for new connections within a minute, without a restart. If the new files don't hold a certificate and its key, the key is too short or other users can access the key file, it keeps the old one and logs an error. The [metrics](metrics.md) say when each certificate in use expires. They always use plain HTTP.

## Who can read the key

The authority doesn't start if users other than root and its own user can read or change a key file. In Kubernetes, mount the Secrets like this:

```yaml
spec:
  securityContext:
    fsGroup: 65532
  containers:
    - name: authority
      volumeMounts:
        - name: external-api-tls
          mountPath: /etc/phemera/authority/external-api-tls
          readOnly: true
        - name: internal-api-tls
          mountPath: /etc/phemera/authority/internal-api-tls
          readOnly: true
  volumes:
    - name: external-api-tls
      secret:
        secretName: authority-external-api-tls
        defaultMode: 0440
    - name: internal-api-tls
      secret:
        secretName: authority-internal-api-tls
        defaultMode: 0440
```

A Secret's files belong to root. `fsGroup` gives them the group 65532, the user ID of the container image, and `defaultMode` lets only the owner and the group read them. Without `defaultMode`, every user in the container can read the key.

Outside Kubernetes, give each key file to the user the authority runs as, with mode 0600 or 0400. If root owns the file, mode 0640 or 0440 works too, and the authority reads it through the file's group. Note that the authority doesn't check who else is in that group. The certificate file can have any mode.

## The internal API's certificate

No component calls the internal API yet, so nothing checks its certificate. Until one does, a self-signed certificate is enough. The authority still checks its key's length and who can read the key file.

In Kubernetes with cert-manager, a self-signed Issuer can create the Secret that the example above mounts:

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: authority-internal-api
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: authority-internal-api
spec:
  secretName: authority-internal-api-tls
  issuerRef:
    name: authority-internal-api
    kind: Issuer
  privateKey:
    algorithm: ECDSA
    size: 256
  dnsNames:
    - localhost
```

Without cert-manager, create the files with openssl:

```sh
openssl req -x509 -newkey ec -pkeyopt ec_paramgen_curve:P-256 -noenc \
    -days 365 -subj /CN=localhost -keyout tls.key -out tls.crt
chmod 0600 tls.key
```

In Kubernetes, `kubectl create secret tls authority-internal-api-tls --cert=tls.crt --key=tls.key` then stores them in the Secret.

This changes when gateways and workers start to call the internal API, because they will check its certificate. They will also authenticate to the authority with their own certificates, over mutual TLS, as [TEP-0002](https://github.com/phemera-dev/enhancements/tree/main/teps/0002-architecture) describes. That's why the internal API has no setting to serve plain HTTP: a proxy that terminates TLS would hide their certificates.
