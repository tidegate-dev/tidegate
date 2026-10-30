# Running under systemd

If you run the authority as a systemd service, give its unit `Type=notify`:

```ini
[Service]
Type=notify
ExecStart=/usr/local/bin/authority
```

systemd then counts the authority as started once it accepts connections, not as soon as the process runs. `systemctl start` waits until then, and units ordered after the authority don't start too early. Outside systemd, the authority starts and stops as usual.
