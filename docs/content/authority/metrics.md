# Metrics

The authority serves its metrics at `/metrics` on their own address, in the Prometheus text format:

```sh
TIDEGATE_AUTHORITY__METRICS__LISTEN_ADDRESS=:9464 authority
```

That way you can make the API public and keep the metrics internal. By default, the metrics only listen on `127.0.0.1`, so set the address if Prometheus scrapes them from another host. The metrics always use plain HTTP and need no login, so only let Prometheus reach their address.

The authority exports the metrics of the Go runtime, like its memory and its goroutines. With [TLS](tls.md), `tidegate_tls_certificate_expiry_seconds` says when the certificate in use expires, as a Unix time, so you can alert before it does.
