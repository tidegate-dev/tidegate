# Running under systemd

If you run the authority as a systemd service, give its unit `Type=notify`:

```ini
[Service]
Type=notify
ExecStart=/usr/local/bin/authority
```

systemd then counts the authority as started once it accepts connections, not as soon as the process runs. `systemctl start` waits until then, and units ordered after the authority don't start too early. Outside systemd, the authority starts and stops as usual.

While the authority [waits for the database](database.md), it isn't started yet. If that takes longer than the unit's `TimeoutStartSec=`, 90 seconds by default, systemd stops the authority and the unit fails. If your database can take longer to start, raise `TimeoutStartSec=`, or set it to `infinity` to wait as long as it takes.
