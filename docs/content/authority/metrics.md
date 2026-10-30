# Metrics

The authority serves its metrics at `/metrics` on their own address, in the Prometheus text format:

```sh
TIDEGATE_AUTHORITY__METRICS__LISTEN_ADDRESS=:9464
```

That way you can make the API public and keep the metrics internal. By default, the metrics only listen on `127.0.0.1`, so set the address if Prometheus scrapes them from another host. The metrics always use plain HTTP.

The authority exports the metrics of the Go runtime, like its memory and its goroutines.
