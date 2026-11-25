---
title: Run Mavis with systemd
description: Serve a read-only content tree on loopback as the mavis user.
weight: 40
---

On a Linux host with systemd, unpack and [verify the release](../getting-started/installation.md) for your CPU. From the extracted directory, install the binary and provision the account (skip account creation if `mavis` already exists and is a dedicated service user):

```sh
sudo install -m 0755 mavis /usr/local/bin/mavis
sudo useradd --system --user-group --no-create-home --home-dir /nonexistent --shell /usr/sbin/nologin mavis
sudo install -d -o root -g mavis -m 0750 /srv/mavis/docs
sudo install -o root -g mavis -m 0640 testdata/docs/README.md /srv/mavis/docs/README.md
```

Give `mavis` read access to your files and traversal permission on their directories. Keep them and `.mavis.conf` writable only by the publisher. Save this as `/etc/systemd/system/mavis.service`:

```ini
[Unit]
Description=Mavis documentation server
After=network.target

[Service]
User=mavis
Group=mavis
WorkingDirectory=/srv/mavis/docs
ExecStart=/usr/local/bin/mavis serve /srv/mavis/docs --host 127.0.0.1 --port 3344
Restart=on-failure
RestartSec=3s
KillSignal=SIGINT
UMask=0077
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes

[Install]
WantedBy=multi-user.target
```

```sh
sudo systemd-analyze verify /etc/systemd/system/mavis.service
sudo systemctl daemon-reload
sudo systemctl enable --now mavis
systemctl status mavis
ps -o user= -p "$(systemctl show -p MainPID --value mavis)"   # mavis
curl --fail http://127.0.0.1:3344/
journalctl -u mavis -b --no-pager
```

For `permission denied`, check every parent with `namei -l /srv/mavis/docs/README.md`. Keep the content out of `/home` when using `ProtectHome=yes`. If the port is in use, change the unit's `--port`. For optional DOCX downloads, make `pandoc` available on the **service's** trusted `PATH` (use `Environment=PATH=...` with absolute directories). After unit changes, run `daemon-reload` and `systemctl restart mavis`.

To remove the service, run `sudo systemctl disable --now mavis`, remove just `/etc/systemd/system/mavis.service`, and run `sudo systemctl daemon-reload`. Keep the content; remove the dedicated account only if nothing else uses it. For an HTTPS deployment, see [Caddy over a Unix socket](linux-caddy.md).
