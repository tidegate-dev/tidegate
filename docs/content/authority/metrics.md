# Metrics

The authority serves its metrics at `/metrics` on their own address, in the Prometheus text format:

```sh
PHEMERA_AUTHORITY__METRICS__LISTEN_ADDRESS=:9464 authority
```

That way you can make the external API public and keep the metrics internal. By default, the metrics only listen on `127.0.0.1`, so set the address if Prometheus scrapes them from another host. The metrics always use plain HTTP and need no login, so keep their address internal. Prometheus scrapes it, and Kubernetes or a load balancer asks the [health checks](health.md) there.

The authority exports the metrics of the Go runtime, like its memory and its goroutines. With [TLS](tls.md), `phemera_tls_certificate_expiry_seconds` says when each certificate in use expires, as a Unix time, so you can alert before it does. Its `cert_file` label holds the certificate's path.
