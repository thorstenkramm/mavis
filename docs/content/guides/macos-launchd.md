---
title: Run Mavis with launchd
description: Install a system LaunchDaemon that serves documentation as the mavis user.
weight: 50
---

Use a system LaunchDaemon for a server that starts at boot without a user login. Have your Mac administrator provision a local, non-admin service account named `mavis`, with no interactive login, shell `/usr/bin/false` and home `/var/empty`. Use your normal account provisioning tool to allocate a **unique** UID and primary GID; never copy numeric IDs from another Mac. Confirm `id mavis` returns the intended identity before continuing.

From the [verified release](../getting-started/installation.md) directory, install the executable and a sample page. Keep the documentation publisher-owned and readable by the service. Give it a separate writable log directory:

```sh
sudo install -d -m 0755 /usr/local/bin /Library/Mavis
sudo install -m 0755 mavis /usr/local/bin/mavis
sudo install -d -o root -g "$(id -gn mavis)" -m 0750 /Library/Mavis/docs
sudo install -o root -g "$(id -gn mavis)" -m 0640 testdata/docs/README.md /Library/Mavis/docs/README.md
sudo install -d -o mavis -g "$(id -gn mavis)" -m 0750 /Library/Logs/Mavis
```

Save the following as `io.system42.mavis.plist`. Each argument is a separate string; launchd does not run a shell. Include a trusted directory containing `pandoc` in the daemon's `PATH` if DOCX export is needed.

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>io.system42.mavis</string>
  <key>UserName</key><string>mavis</string>
  <key>ProgramArguments</key><array>
    <string>/usr/local/bin/mavis</string><string>serve</string><string>/Library/Mavis/docs</string>
    <string>--host</string><string>127.0.0.1</string><string>--port</string><string>3344</string>
  </array>
  <key>WorkingDirectory</key><string>/Library/Mavis/docs</string>
  <key>EnvironmentVariables</key><dict><key>PATH</key><string>/usr/local/bin:/usr/bin:/bin</string></dict>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>/Library/Logs/Mavis/stdout.log</string>
  <key>StandardErrorPath</key><string>/Library/Logs/Mavis/stderr.log</string>
</dict></plist>
```

```sh
plutil -lint io.system42.mavis.plist
plist=/Library/LaunchDaemons/io.system42.mavis.plist
sudo install -o root -g wheel -m 0644 io.system42.mavis.plist "$plist"
sudo launchctl bootstrap system "$plist"
sudo launchctl print system/io.system42.mavis
ps -u mavis -o user,pid,command
curl --fail http://127.0.0.1:3344/
sudo tail /Library/Logs/Mavis/stderr.log
```

If launch fails, check `StandardErrorPath`, verify that `mavis` can traverse and read the content directory, and check whether TCP 3344 is occupied. LaunchDaemons do not inherit your shell's PATH. After editing the plist, use `sudo launchctl bootout system/io.system42.mavis` followed by `bootstrap` to reread it. To uninstall, boot it out and remove only the installed plist; retain your documents. Retire the service account through your administrator if it is no longer used.
