---
title: Run Mavis with Servy on Windows
description: Register Mavis as a least-privilege Windows service listening on loopback TCP.
weight: 60
---

Use [Servy](https://servy-win.github.io/) to wrap the console executable. These settings follow the [Servy 11.0 desktop app and manager](https://github.com/aelassas/servy/wiki/Overview). Use an elevated administrator session to install and manage the service.

1. Extract the [verified Windows archive](../getting-started/installation.md). Install `mavis.exe` in `C:\Program Files\Mavis`. Create `C:\ProgramData\Mavis\Team Docs` for Markdown and `C:\ProgramData\Mavis\Logs` for output. Place a `README.md` in the docs directory.
2. Install Servy from its official distribution. In the desktop app create a service named `Mavis` with these values:

   | Field | Value |
   | --- | --- |
   | Executable path | `C:\Program Files\Mavis\mavis.exe` |
   | Startup directory | `C:\ProgramData\Mavis` |
   | Parameters | `serve "C:\ProgramData\Mavis\Team Docs" --host 127.0.0.1 --port 3344` |
   | Startup type | Automatic |
   | Log On | `NT AUTHORITY\LocalService`, no password |
   | stdout / stderr logs | `C:\ProgramData\Mavis\Logs\stdout.log` / `C:\ProgramData\Mavis\Logs\stderr.log` |

3. Grant LocalService **Read & execute** on the binary, **Read & execute** on the content tree (including directories), and **Modify** on the log directory only. Keep executable and content write permissions with administrators/publishers. Servy's own vault permissions do not grant access to your documentation. Configure an absolute, trusted directory on the service's PATH for optional pandoc; Windows uses TCP rather than a Unix socket.
4. Install the service from Servy, then verify in PowerShell:

   ```powershell
   Start-Service Mavis
   Get-CimInstance Win32_Service -Filter "Name='Mavis'" | Select-Object State,StartName,PathName
   (Invoke-WebRequest -UseBasicParsing http://127.0.0.1:3344/).StatusCode  # 200
   Get-Content 'C:\ProgramData\Mavis\Logs\stderr.log' -Tail 20
   ```

If startup fails, check Servy Manager's service logs, the quoted directory argument, service-account permissions and whether port 3344 is free. To update the binary, stop the service before replacement and start it again. To remove, run `Stop-Service Mavis`, choose **Uninstall** for `Mavis` in Servy Manager and verify `Get-Service Mavis -ErrorAction SilentlyContinue` returns no service. Keep the content and any logs you need.
