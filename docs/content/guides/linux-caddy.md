---
title: Serve HTTPS through Caddy on Linux
description: Put Caddy in front of a group-owned Mavis socket with a hardened systemd unit.
weight: 70
---

This Linux example serves `https://mavis.example.com`. Substitute a name you control, point its A/AAAA records to Caddy and allow inbound TCP 80 and 443 for automatic HTTPS. TLS encrypts traffic; Mavis has no login. Restrict access in Caddy or at the network boundary if the documents are private.

Follow [Linux account and binary setup](linux-systemd.md), and install Caddy 2 with its `caddy` service account. Stop any previous Mavis instance. Create a local group for proxy clients, then add the Caddy user:

```sh
sudo groupadd --system mavis-proxy
sudo usermod -aG mavis-proxy caddy
```

In `/srv/mavis/docs/.mavis.conf` use only this listener (remove `host` and `port`):

```toml
[server]
socket = "/run/mavis/mavis.sock"
socket_group = "mavis-proxy"
```

Keep the content and configuration publisher-owned, readable by `mavis`, and private from proxy users. Install this complete `/etc/systemd/system/mavis.service` **instead of** the TCP unit:

```ini
[Unit]
Description=Mavis documentation backend for Caddy

[Service]
User=mavis
Group=mavis-proxy
SupplementaryGroups=mavis
RuntimeDirectory=mavis
RuntimeDirectoryMode=0750
WorkingDirectory=/srv/mavis/docs
ExecStart=/usr/local/bin/mavis serve /srv/mavis/docs
Restart=on-failure
RestartSec=3s
KillSignal=SIGINT
UMask=0077
NoNewPrivileges=yes
CapabilityBoundingSet=
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
PrivateDevices=yes
RestrictAddressFamilies=AF_UNIX

[Install]
WantedBy=multi-user.target
```

Systemd creates the writable runtime directory as `mavis:mavis-proxy`; group members can traverse but not replace the socket. The listener is `0660`. Never make it world-writable. In `/etc/caddy/Caddyfile`, add:

```caddyfile
mavis.example.com {
    reverse_proxy unix//run/mavis/mavis.sock
}
```

```sh
sudo systemd-analyze verify /etc/systemd/system/mavis.service
sudo systemctl daemon-reload
sudo systemctl enable mavis
sudo systemctl restart mavis
sudo caddy validate --config /etc/caddy/Caddyfile --adapter caddyfile
sudo systemctl restart caddy  # pick up supplementary membership, not just the config
stat -c '%U:%G %a %n' /run/mavis /run/mavis/mavis.sock
sudo -u caddy curl --fail --unix-socket /run/mavis/mavis.sock http://localhost/
sudo -u nobody curl --unix-socket /run/mavis/mavis.sock http://localhost/  # must fail
sudo ss -lntp | grep mavis  # expect no Mavis TCP listener
curl --fail https://mavis.example.com/
```

A 502 usually means a missing socket, wrong directory permissions, a group change not yet active in Caddy, or a host security policy denying the connection. Check `namei -l /run/mavis/mavis.sock` and `journalctl -u mavis -u caddy -b`. For a certificate error check DNS, firewall and Caddy's journal. On an isolated test host use Caddy's `tls internal` and trust its test CA in the test client only.

To remove, delete just the Mavis site block and validate/reload Caddy. Disable and remove the Mavis unit as in the Linux guide; systemd removes its runtime directory. Remove Caddy's membership with `sudo gpasswd -d caddy mavis-proxy`, restart Caddy, and delete the dedicated group if unused. Remove the socket settings before starting Mavis in TCP mode again.
