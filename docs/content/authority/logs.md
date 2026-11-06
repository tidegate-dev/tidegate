# Logs

The authority writes its log to standard error, one JSON object per line. It logs one line for each call to its API:

```json
{"time":"2026-11-06T13:10:33.095136171+01:00","level":"INFO","msg":"handled a call","procedure":"/tidegate.authority.v1.LoginService/Login","code":"unauthenticated","duration":94319689,"peer":"127.0.0.1:46374"}
```

If you read the log yourself instead of a log collector, set `TIDEGATE_AUTHORITY__LOG__FORMAT` to `text`. If you don't want a line for every call, set `TIDEGATE_AUTHORITY__LOG__LEVEL` to `warn`. This also leaves out the other lines at info level. The [configuration](config.md) page lists both settings.

A line for a call never has the body of the request or the response. A failed login also logs the user name it was given, but the authority never logs a password or a login token.

Some requests get no line, because the authority refuses them before the call starts. That happens when a request is too large, goes to a path that isn't an API method, isn't valid Connect, gRPC or gRPC-Web, or has a message that can't be decoded.

Use these lines to find failing and slow calls. They don't record who did what, so they don't replace an audit log.

| Field | Meaning |
|---|---|
| `procedure` | The API method that was called |
| `code` | The [Connect code](https://connectrpc.com/docs/protocol/#error-codes) the client got, or `ok` |
| `duration` | How long the authority took to answer. In JSON it's in nanoseconds, in text it's written like `94.3ms`. |
| `peer` | The client's address and port, or the proxy's if a proxy sits in front of the authority |
