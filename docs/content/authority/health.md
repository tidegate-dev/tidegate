# Health checks

The authority answers two health checks on the address of its [metrics](metrics.md). Like the metrics, they use plain HTTP and need no login, so Kubernetes, a load balancer or a monitoring system can ask them without TLS or a token.

| Path | Answers `200 OK` when | Otherwise |
|---|---|---|
| `/healthz` | The authority runs | No answer |
| `/readyz` | The authority's database answers within a second | `503 Service Unavailable` |

In Kubernetes, point the liveness probe at `/healthz` and the readiness probe at `/readyz`:

```yaml
ports:
  - name: metrics
    containerPort: 9464
livenessProbe:
  httpGet:
    path: /healthz
    port: metrics
readinessProbe:
  httpGet:
    path: /readyz
    port: metrics
```

Kubernetes probes the pod's IP address, so the metrics must not listen only on `127.0.0.1`. The container image sets `PHEMERA_AUTHORITY__METRICS__LISTEN_ADDRESS` to `:9464`.

Don't use `/readyz` for the liveness probe. When the database is down, Kubernetes would restart every authority, and that doesn't bring the database back.

The authority opens its addresses only after it has connected to the database, and migrated it if `PHEMERA_AUTHORITY__DATABASE__MIGRATE_ON_START` is `true`. Until then, both checks fail. If a migration can take longer than your liveness probe allows, add a startup probe on `/healthz` with a higher `failureThreshold`.

`/readyz` borrows a connection from the authority's connection pool. So it also fails when every connection stays busy for that second.

## Logs

The health checks get no line in the [log](logs.md). When `/readyz` starts to fail, the authority logs a warning with the database's error. When it answers `200 OK` again, the authority logs that at info level.
