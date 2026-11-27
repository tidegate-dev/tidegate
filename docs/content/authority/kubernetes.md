# Running in Kubernetes

This page shows how to run the authority in a Kubernetes cluster, with a [CloudNativePG](https://cloudnative-pg.io/) database, a certificate from [cert-manager](https://cert-manager.io/) and a proxy that terminates TLS. The manifests in `dev/kind/` are only for development: they connect to the database without TLS and run the authority without a security context.

The examples use the namespace `phemera`, the database cluster `phemera-db` and the host name `phemera.example.com`.

## Image

phemera publishes no image yet. Build one from `authority/Containerfile` in the root of the repository, and push it to your registry:

```sh
podman build --file authority/Containerfile --tag <registry>/phemera-authority:<tag> .
podman push <registry>/phemera-authority:<tag>
```

Docker takes the same arguments. The image runs `/authority` as the user 65532. It listens for the external API on port 8080 and for the metrics on port 9464, both on every interface. The internal API keeps its default address, `127.0.0.1:8081`.

## Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: phemera
  labels:
    pod-security.kubernetes.io/enforce: restricted
```

The label makes Kubernetes refuse every Pod in the namespace that doesn't meet the [restricted Pod Security Standard](https://kubernetes.io/docs/concepts/security/pod-security-standards/#restricted). The authority's Pod below meets it, and so do the Pods of CloudNativePG.

## Database

```yaml
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: phemera-db
  namespace: phemera
spec:
  instances: 3
  bootstrap:
    initdb:
      database: phemera
      owner: phemera
  storage:
    size: 1Gi
```

CloudNativePG creates the database `phemera` and a role of the same name that owns it. It keeps the role's password in the Secret `phemera-db-app`, and the CA that signs the server's certificate in the Secret `phemera-db-ca`. The Service `phemera-db-rw` always points to the primary.

The authority connects with this URL:

```
postgres://phemera@phemera-db-rw.phemera.svc:5432/phemera?sslmode=verify-full&sslrootcert=/etc/phemera/db-ca/ca.crt
```

With `sslmode=verify-full`, the authority checks the server's certificate against the CA in `sslrootcert`, and the host name against the certificate. The URL holds no password. The authority reads the standard Postgres variables, like `PGPASSWORD`, so the Deployment below sets `PGPASSWORD` from the Secret `phemera-db-app`. That way, the password isn't in the Deployment.

The Deployment mounts only the file `ca.crt` of the Secret `phemera-db-ca`, with `items`. The Secret also holds the CA's private key. The authority doesn't need it, and anyone who could read it in the container could sign a certificate that the authority trusts.

Note that the file in `sslrootcert` must exist when the authority starts. The authority checks the URL along with its other settings, and that reads the file. If the file is missing, the authority stops and reports that the URL isn't a valid Postgres connection URL, although the URL itself is fine.

The authority also stops when it can't reach the database. On a first install, it may restart a few times until CloudNativePG has created the database.

The authority stores the passwords and private keys of SSH assets in this database without encrypting them, so limit who can read the database and its backups. The page on the [database](database.md) explains this.

### Migrations

The Deployment sets `PHEMERA_AUTHORITY__DATABASE__MIGRATE_ON_START` to `true`, so each new version migrates the database when it starts. This works with the role that CloudNativePG creates: it owns the database, so it may change the schema and create the schema `secret_engine`. To migrate in a separate step instead, run `authority migrate` before you start a new version, as the page on the [database](database.md) explains.

## Internal API

Nothing calls the internal API yet, so keep it on its default address, `127.0.0.1:8081`, where nothing outside the Pod reaches it. It still serves HTTPS, and the authority doesn't start without its certificate. While nothing calls it, a self-signed certificate is enough:

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: selfsigned
  namespace: phemera
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: authority-internal-api
  namespace: phemera
spec:
  secretName: authority-internal-api-tls
  issuerRef:
    name: selfsigned
    kind: Issuer
  privateKey:
    algorithm: ECDSA
    size: 256
  usages:
    - server auth
  dnsNames:
    - localhost
  ipAddresses:
    - 127.0.0.1
```

cert-manager renews the certificate before it expires, and the authority uses the new one without a restart. Once a component calls the internal API, the certificate has to come from a CA that the component trusts.

## External API behind a proxy

In this setup, a proxy terminates TLS, for example a [Gateway API](https://gateway-api.sigs.k8s.io/) implementation based on Envoy. The authority serves plain HTTP, because the Deployment sets `PHEMERA_AUTHORITY__EXTERNAL_API__TLS__DISABLED` to `true`. It logs a warning about this when it starts.

Without TLS, the external API serves HTTP/1.1 and h2c, which is HTTP/2 without TLS. gRPC clients need HTTP/2, so the Service tells the proxy to use h2c with `appProtocol`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: authority
  namespace: phemera
  labels:
    app.kubernetes.io/name: authority
spec:
  selector:
    app.kubernetes.io/name: authority
  ports:
    - name: http
      port: 8080
      targetPort: http
      appProtocol: kubernetes.io/h2c
    - name: metrics
      port: 9464
      targetPort: metrics
---
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: authority
  namespace: phemera
spec:
  parentRefs:
    - name: <gateway>
      namespace: <gateway-namespace>
  hostnames:
    - phemera.example.com
  rules:
    - backendRefs:
        - name: authority
          port: 8080
```

`<gateway>` and `<gateway-namespace>` name your Gateway. Its listener for `phemera.example.com` must serve HTTPS and allow routes from the namespace `phemera`.

Note that the connection from the proxy to the authority is unencrypted, including the passwords and tokens of logins. This NetworkPolicy lets only the proxy reach the external API, and only Prometheus reach the metrics:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: authority
  namespace: phemera
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: authority
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: envoy-gateway-system
      ports:
        - port: 8080
    - from:
        - namespaceSelector:
            matchLabels:
              kubernetes.io/metadata.name: monitoring
      ports:
        - port: 9464
```

It expects the proxy in the namespace `envoy-gateway-system` and Prometheus in `monitoring`, so change the names to match your cluster. A NetworkPolicy only works if your network plugin enforces it. It keeps other Pods away from the port, but it doesn't encrypt the traffic. If the proxy runs on another node, the traffic crosses the network in plain text, unless your network plugin encrypts it.

### When the authority serves TLS itself

To encrypt the connection all the way to the authority, leave out `PHEMERA_AUTHORITY__EXTERNAL_API__TLS__DISABLED` and give the external API its own certificate, in the Secret `authority-external-api-tls`. Mount it at `/etc/phemera/authority/external-api-tls` like the internal API's certificate, with `defaultMode: 0440`, as the page on [TLS](tls.md) shows. Remove `appProtocol: kubernetes.io/h2c` from the Service, because it means HTTP/2 without TLS. The proxy then either passes the TLS connection through to the authority, or opens a TLS connection of its own to the authority and checks its certificate.

## Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: authority
  namespace: phemera
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: authority
  template:
    metadata:
      labels:
        app.kubernetes.io/name: authority
        app.kubernetes.io/part-of: phemera
    spec:
      automountServiceAccountToken: false
      securityContext:
        runAsNonRoot: true
        runAsUser: 65532
        runAsGroup: 65532
        fsGroup: 65532
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: authority
          image: <registry>/phemera-authority:<tag>
          env:
            - name: PHEMERA_AUTHORITY__EXTERNAL_API__TLS__DISABLED
              value: "true"
            - name: PHEMERA_AUTHORITY__DATABASE__URL
              value: postgres://phemera@phemera-db-rw.phemera.svc:5432/phemera?sslmode=verify-full&sslrootcert=/etc/phemera/db-ca/ca.crt
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: phemera-db-app
                  key: password
            - name: PHEMERA_AUTHORITY__DATABASE__MIGRATE_ON_START
              value: "true"
          ports:
            - name: http
              containerPort: 8080
            - name: metrics
              containerPort: 9464
          readinessProbe:
            tcpSocket:
              port: http
          livenessProbe:
            tcpSocket:
              port: http
            initialDelaySeconds: 30
            periodSeconds: 30
          resources:
            requests:
              cpu: 100m
              memory: 64Mi
            limits:
              memory: 768Mi
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop:
                - ALL
          volumeMounts:
            - name: internal-api-tls
              mountPath: /etc/phemera/authority/internal-api-tls
              readOnly: true
            - name: db-ca
              mountPath: /etc/phemera/db-ca
              readOnly: true
      volumes:
        - name: internal-api-tls
          secret:
            secretName: authority-internal-api-tls
            defaultMode: 0440
        - name: db-ca
          secret:
            secretName: phemera-db-ca
            items:
              - key: ca.crt
                path: ca.crt
```

The [configuration](config.md) page lists every setting.

### Pod security

| Setting | Why |
|---|---|
| `runAsNonRoot`, `runAsUser` and `runAsGroup` | 65532 is the image's user. With `runAsNonRoot`, Kubernetes refuses to start the container as root. |
| `fsGroup: 65532` | Gives the files of the mounted Secrets the group 65532. With `defaultMode: 0440`, the authority can read its key and other users can't. The authority refuses a key that other users can read. |
| `seccompProfile` | Allows only the system calls in the container runtime's default profile. |
| `allowPrivilegeEscalation: false` and `capabilities.drop` | The authority needs no privileges. |
| `readOnlyRootFilesystem: true` | The authority writes no files. Its log goes to standard error. |
| `automountServiceAccountToken: false` | The authority doesn't call the Kubernetes API, so it gets no token for it. |

### Probes

The authority has no health endpoint yet, so both probes open a TCP connection to the external API. The authority opens its ports only after it has connected to the database and migrated it, so the readiness probe passes once the authority can answer calls. Note that the probes don't notice when the database goes away later.

### Resources

argon2id hashes each password with 64 MiB of memory. The authority checks at most four passwords at once, so logins use up to 256 MiB. Go's garbage collector lets the heap grow to about twice the memory in use before it frees any, so four logins at once can take 512 MiB. The limit of 768 MiB leaves room for that and the rest of the authority. When you run `authority create-initial-admin` with `kubectl exec`, it runs in the same container and hashes a password there too. An idle authority needs much less, so the request is lower.

The Deployment sets no CPU limit, because argon2id uses four threads for each password, and a limit would slow down logins.

## Metrics

The metrics use plain HTTP and need no login, so the NetworkPolicy above lets only Prometheus reach their port. The page on [metrics](metrics.md) lists what they hold. With the Prometheus Operator, a ServiceMonitor scrapes them:

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: authority
  namespace: phemera
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: authority
  endpoints:
    - port: metrics
```

## First admin

Once the authority is ready, its database is migrated. Create the first admin in the authority's container:

```sh
kubectl exec -it -n phemera deploy/authority -- /authority create-initial-admin alice
```

The command gets the container's variables, so it connects to the database like the authority. Keep `-it`: the page on the [first admin](initial-admin.md) explains why. Then [sign in](../cli/sign-in.md) with the CLI:

```sh
phemera login https://phemera.example.com
```
